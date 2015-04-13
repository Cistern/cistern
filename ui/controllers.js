var cisternURL = 'http://localhost:8080'
var flowURL = 'http://localhost:8080/devices/127.0.0.1/flows'

function NavigationCtrl($scope, $location) {
  $scope.isActive = function(viewLocation) {
    return viewLocation === $location.path();
  }
}

function DevicesCtrl($scope, $http) {
  $http.get(cisternURL+'/devices/').then(function(response) {
    if (response.status == 200) {
      $scope.devices = response.data.data;

      $scope.devices.sort(function(a, b) {
        if (a.ip < b.ip) {
          return -1;
        }

        if (a.ip > b.ip) {
          return 1;
        }

        return 0;
      });
    }
  });

  $scope.loadMetrics = function(device) {
    $http.get(cisternURL + '/devices/' + device.ip + '/metrics').then(function(response) {
      if (response.status == 200) {
        device.metrics = response.data.data;
        device.metrics.sort(function(a, b) {
          if (a.name < b.name) {
            return -1;
          }

          if (a.name > b.name) {
            return 1;
          }

          return 0;
          });
      }
    });
  }
}

function FlowsCtrl($scope, $http) {
  $http.get(flowURL).then(function(response) {
    if (response.status == 200) {
      $scope.flows = response.data.data;     
    }
  });
}
