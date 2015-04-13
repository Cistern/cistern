var cisternApp = angular.module('cisternUI', [
  'ngRoute'
])

.config(
  function ($routeProvider) {
    $routeProvider.when('/', {
      redirectTo: '/devices'
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
.controller('FlowsCtrl', FlowsCtrl)
.controller('NavigationController', NavigationCtrl)

.filter('flowOrdering', function() {
  return function(input) {
    switch(input) {
    case 'byBytes':
      return 'by bytes';
    case 'byPackets':
      return 'by packets';
    }

    return "";
  };
});