'use strict';

/* Controllers */

angular.module('photoshare.controllers', ['photoshare.services'])
    .controller('AppCtrl', ['$scope',
                            '$location',
                            '$timeout',
                            'Session',
                            'Authenticator',
                            'Alert',
                            function ($scope,
                                      $location,
                                      $timeout,
                                      Session,
                                      Authenticator,
                                      Alert) {

            $scope.session = Session;
            $scope.alert = Alert;
            $scope.searchQuery = "";

            Authenticator.init();

            $scope.$watch('alert.message', function (newValue, oldValue) {
                if (newValue) {
                    $timeout(function () { Alert.dismiss(); }, 3000);
                }
            });

            $scope.logout = function () {
                Authenticator.logout().then(function () {
                    $location.path("/list");
                });
            };

            $scope.login = function () {
                Session.setLastLoginUrl();
                $location.path("/login");
            };

            $scope.doSearch = function () {
                $location.path("/search/" + $scope.searchQuery);
                $scope.searchQuery = "";
            };
        }])

    .controller('ListCtrl', ['$scope',
                             '$location',
                             '$routeParams',
                             'Photo',
                             'pageSize',
                             function ($scope,
                                       $location,
                                       $routeParams,
                                       Photo,
                                       pageSize) {
            var page = 1,
                stopScrolling = false,
                q = $routeParams.q || "",
                ownerID = $routeParams.ownerID || "",
                ownerName = $routeParams.ownerName || "";

            $scope.photos = [];
            $scope.searchQuery = q;
            $scope.ownerName = ownerName;
            $scope.nextPage = function () {
                if (!stopScrolling) {
                    Photo.query({page: page, q: q, ownerID: ownerID}).$promise.then(function (photos) {
                        $scope.photos = $scope.photos.concat(photos);
                        if (photos.length < pageSize) {
                            stopScrolling = true;
                        }
                    });
                }
                page += 1;
            };

            $scope.getDetail = function (photo) {
                $location.path("/detail/" + photo.id);
            };

        }])

    .controller('DetailCtrl', ['$scope',
                               '$routeParams',
                               '$location',
                               'Photo',
                               'Tag',
                               'Session',
                               'Alert',
                               function ($scope,
                                         $routeParams,
                                         $location,
                                         Photo,
                                         Tag,
                                         Session,
                                         Alert) {

            function doUpdate(onSuccess) {
                var taglist = $scope.photo.taglist || "";
                if (!taglist) {
                    $scope.photo.tags = [];
                } else {
                    $scope.photo.tags = taglist.trim().split(" ");
                }
                Photo.update({id: $scope.photo.id,
                              title: $scope.photo.title,
                              tags: $scope.photo.tags}, function () {
                    onSuccess();
                });
            }

            $scope.photo = null;
            $scope.editTitle = false;
            $scope.editTags = false;

            Photo.get({id: $routeParams.id}).$promise.then(function (photo) {
                $scope.photo = photo;
                $scope.photo.taglist = $scope.photo.tags ? $scope.photo.tags.join(" ") : "";
            });
            $scope.deletePhoto = function () {
                if (!$scope.photo.canDelete) {
                    return;
                }
                $scope.photo.$delete(function () {
                    Alert.warning('Your photo has been deleted');
                    $location.path("/");
                });
            };
            $scope.showEditForm = function () {
                if ($scope.photo.canEdit) {
                    $scope.editTitle = true;
                }
            };
            $scope.hideEditForm = function () {
                $scope.editTitle = false;
            };
            $scope.showEditTagsForm = function () {
                if ($scope.photo.canEdit) {
                    $scope.editTags = true;
                }
            };
            $scope.hideEditTagsForm = function () {
                $scope.editTags = false;
            };
            $scope.updateTitle = function () {
                doUpdate(function () { $scope.editTitle = false; });
            };
            $scope.updateTags = function () {
                doUpdate(function () {
                    $scope.editTags = false;
                });
            };

        }])

    .controller('TagsCtrl', ['$scope',
                             '$location',
                             'Tag', function ($scope, $location, Tag) {
        $scope.tags = [];
        $scope.orderField = 'name';

        Tag.query().$promise.then(function (tags) {
            $scope.tags = tags;
        });

        $scope.doSearch = function (tag) {
            $location.path("/search/" + tag);
        };

        $scope.orderTags = function (field) {
            $scope.orderField = field;
        };

    }])

    .controller('UploadCtrl', ['$scope',
                               '$location',
                               'Session',
                               'Alert',
                               'Photo', function ($scope,
                                                  $location,
                                                  Session,
                                                  Alert,
                                                  Photo) {
        Session.check();
        $scope.newPhoto = new Photo();
        $scope.upload = null;
        $scope.formDisabled = false;
        $scope.uploadPhoto = function () {
            $scope.formDisabled = true;
            var taglist = $scope.newPhoto.taglist || "";
            if (!taglist) {
                $scope.newPhoto.tags = [];
            } else {
                $scope.newPhoto.tags = taglist.trim().split(" ");
            }
            $scope.newPhoto.$save(
                function () {
                    $scope.newPhoto = new Photo();
                    Alert.success('Your photo has been uploaded');
                    $location.path("/list");
                },
                function () {
                    $scope.formDisabled = false;
                }
            );
        };

    }])

    .controller('LoginCtrl', ['$scope',
                              '$location',
                              'Session',
                              'Authenticator',
                              'Alert',
                              'authToken', function ($scope,
                                                     $location,
                                                     Session,
                                                     Authenticator,
                                                     Alert,
                                                     authToken) {

        $scope.loginCreds = new Authenticator.resource();
        $scope.login = function () {
            $scope.loginCreds.$save(function (result, headers) {
                $scope.loginCreds = new Authenticator.resource();
                if (result.loggedIn) {
                    Authenticator.login(result, headers(authToken));
                    Alert.success("Welcome back, " + result.name);
                    var path = Session.getLastLoginUrl() || "/list";
                    $location.path(path);
                }
            });
        };
    }])

    .controller('SignupCtrl', ['$scope',
                               '$location',
                               'User',
                               'Authenticator',
                               'Alert',
                               'authToken', function ($scope,
                                                      $location,
                                                      User,
                                                      Authenticator,
                                                      Alert,
                                                      authToken) {

        $scope.newUser = new User();
        $scope.signup = function () {
            $scope.newUser.$save(function (result, headers) {
                Authenticator.login(result, headers(authToken));
                $scope.newUser = new User();
                Alert.success("Welcome, " + result.name);
                $location.path("/list");
            });
        };
    }]);
