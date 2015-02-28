var cisternURL = 'http://localhost:8080'

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

function FlowsCtrl() {

}
