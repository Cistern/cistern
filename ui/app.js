var cisternApp = angular.module('cisternUI', [
  'ngRoute'
])

.config(
  function ($routeProvider) {
    $routeProvider.when('/', {
      templateUrl: 'partials/devices.html',
      controller: 'DevicesCtrl'
    }).when('/devices', {
      templateUrl: 'partials/devices.html',
      controller: 'DevicesCtrl'
    }).when('/flows', {
      templateUrl: 'partials/flows.html',
      controller: 'FlowsCtrl'
    });
  }
)

.controller('DevicesCtrl', DevicesCtrl)
.controller('FlowsCtrl', FlowsCtrl);
