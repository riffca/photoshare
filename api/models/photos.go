package models

import (
	"database/sql"
	"fmt"
	"github.com/coopernurse/gorp"
	"github.com/danjac/photoshare/api/storage"
	"strings"
	"time"
)

const (
	PageSize = 32
)

type PhotoManager interface {
	Insert(*Photo) error
	Update(*Photo) error
	Delete(*Photo) error
	Get(photoID string) (*Photo, error)
	GetDetail(photoID string) (*PhotoDetail, error)
	GetTagCounts() ([]TagCount, error)
	All(pageNum int64) ([]Photo, error)
	ByOwnerID(pageNum int64, ownerID string) ([]Photo, error)
	Search(pageNum int64, q string) ([]Photo, error)
	UpdateTags(*Photo) error
}

type Tag struct {
	ID   int64  `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

type PhotoTag struct {
	PhotoID int64 `db:"photo_id"`
	TagID   int64 `db:"tag_id"`
}

type TagCount struct {
	Name      string `db:"name" json:"name"`
	Photo     string `db:"photo" json:"photo"`
	NumPhotos int64  `db:"num_photos" json:"numPhotos"`
}

type Photo struct {
	ID        int64     `db:"id" json:"id"`
	OwnerID   int64     `db:"owner_id" json:"ownerId"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	Title     string    `db:"title" json:"title"`
	Photo     string    `db:"photo" json:"photo"`
	Tags      []string  `db:"-" json:"tags"`
}

var photoCleaner = storage.NewPhotoCleaner()

func (photo *Photo) PreInsert(s gorp.SqlExecutor) error {
	photo.CreatedAt = time.Now()
	return nil
}

func (photo *Photo) PreDelete(s gorp.SqlExecutor) error {
	go photoCleaner.Clean(photo.Photo)
	return nil
}

func (photo *Photo) CanDelete(user *User) bool {
	return user.ID == photo.OwnerID || user.IsAdmin
}

func (photo *Photo) CanEdit(user *User) bool {
	return user.ID == photo.OwnerID
}

type PhotoDetail struct {
	ID        int64     `db:"id" json:"id"`
	OwnerID   int64     `db:"owner_id" json:"ownerId"`
	OwnerName string    `db:"owner_name" json:"ownerName"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	Title     string    `db:"title" json:"title"`
	Photo     string    `db:"photo" json:"photo"`
	Tags      []string  `db:"-" json:"tags"`
}

type defaultPhotoManager struct{}

var photoMgr = &defaultPhotoManager{}

func NewPhotoManager() PhotoManager {
	return photoMgr
}

func (mgr *defaultPhotoManager) Delete(photo *Photo) error {
	t, err := dbMap.Begin()
	if err != nil {
		return err
	}
	if _, err := dbMap.Delete(photo); err != nil {
		return err
	}

	return t.Commit()
}

func (mgr *defaultPhotoManager) Update(photo *Photo) error {
	t, err := dbMap.Begin()
	if err != nil {
		return err
	}
	if _, err := dbMap.Update(photo); err != nil {
		return err
	}
	if err := mgr.UpdateTags(photo); err != nil {
		return err
	}
	return t.Commit()
}

func (mgr *defaultPhotoManager) Insert(photo *Photo) error {
	t, err := dbMap.Begin()
	if err != nil {
		return err
	}
	if err := dbMap.Insert(photo); err != nil {
		return err
	}
	if err := mgr.UpdateTags(photo); err != nil {
		return err
	}
	return t.Commit()
}

func (mgr *defaultPhotoManager) UpdateTags(photo *Photo) error {

	var (
		args    = []string{"$1"}
		params  = []interface{}{interface{}(photo.ID)}
		isEmpty = true
	)
	for i, name := range photo.Tags {
		name = strings.TrimSpace(name)
		if name != "" {
			args = append(args, fmt.Sprintf("$%d", i+2))
			params = append(params, interface{}(strings.ToLower(name)))
			isEmpty = false
		}
	}
	if isEmpty {
		_, err := dbMap.Exec("DELETE FROM photo_tags WHERE photo_id=$1", photo.ID)
		return err
	}
	_, err := dbMap.Exec(fmt.Sprintf("SELECT add_tags(%s)", strings.Join(args, ",")), params...)
	return err

}

func (mgr *defaultPhotoManager) Get(photoID string) (*Photo, error) {

	photo := &Photo{}
	obj, err := dbMap.Get(photo, photoID)
	if err != nil {
		return photo, err
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(*Photo), nil
}

func (mgr *defaultPhotoManager) GetDetail(photoID string) (*PhotoDetail, error) {

	photo := &PhotoDetail{}

	q := "SELECT p.*, u.name AS owner_name " +
		"FROM photos p JOIN users u ON u.id = p.owner_id " +
		"WHERE p.id=$1"

	if err := dbMap.SelectOne(photo, q, photoID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var tags []Tag

	if _, err := dbMap.Select(&tags,
		"SELECT t.* FROM tags t JOIN photo_tags pt ON pt.tag_id=t.id "+
			"WHERE pt.photo_id=$1", photo.ID); err != nil {
		return photo, err
	}
	for _, tag := range tags {
		photo.Tags = append(photo.Tags, tag.Name)
	}

	return photo, nil

}

func (mgr *defaultPhotoManager) ByOwnerID(pageNum int64, ownerID string) ([]Photo, error) {

	var photos []Photo
	if _, err := dbMap.Select(&photos,
		"SELECT * FROM photos WHERE owner_id = $1"+
			"ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		ownerID, PageSize, getOffset(pageNum)); err != nil {
		return photos, err
	}
	return photos, nil

}

func (mgr *defaultPhotoManager) Search(pageNum int64, q string) ([]Photo, error) {

	var photos []Photo

	q = "%" + q + "%"
	if _, err := dbMap.Select(&photos,
		"SELECT DISTINCT p.* FROM photos p "+
			"INNER JOIN users u ON u.id = p.owner_id "+
			"LEFT JOIN photo_tags pt ON pt.photo_id=p.id "+
			"LEFT JOIN tags t ON t.id = pt.tag_id "+
			"WHERE (p.title ILIKE $1 OR u.name ILIKE $1 OR t.name ILIKE $1) "+
			"ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		q, PageSize, getOffset(pageNum)); err != nil {
		return photos, err
	}
	return photos, nil

}

func (mgr *defaultPhotoManager) All(pageNum int64) ([]Photo, error) {

	var photos []Photo

	if _, err := dbMap.Select(&photos,
		"SELECT * FROM photos "+
			"ORDER BY created_at DESC LIMIT $1 OFFSET $2", PageSize, getOffset(pageNum)); err != nil {
		return photos, err
	}
	return photos, nil
}

func (mgr *defaultPhotoManager) GetTagCounts() ([]TagCount, error) {
	var tags []TagCount
	if _, err := dbMap.Select(&tags, "SELECT name, photo, num_photos FROM tag_counts"); err != nil {
		return tags, err
	}
	return tags, nil
}

func getOffset(pageNum int64) int64 {
	return (pageNum - 1) * PageSize
}
