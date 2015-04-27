var cisternURL = 'http://localhost:8080'
var flowURL = 'http://localhost:8080/devices/127.0.0.1/flows'

var templateDefinitions = {
  "CPU":
  {
    "desc": "CPU usage",
    "metrics": ['cpu.idle', 'cpu.user', 'cpu.sys', 'cpu.nice', 'cpu.softintr', 'cpu.intr', 'cpu.wio']
  },

  "Disk":
  {
    "desc": "Disk IO",
    "metrics": ['disk.bytes_written', 'disk.bytes_read'],
    "factors": [-1, 1]
  },

  "Memory":
  {
    "desc": "Memory usage",
    "metrics": ['mem.free', 'mem.shared', 'mem.buffers', 'mem.cached']
  }
};

function NavigationCtrl($scope, $location) {
  $scope.isActive = function(viewLocation) {
    return viewLocation === $location.path();
  }
}

function DevicesCtrl($scope, $http) {
  $scope.charts = [
    "CPU", "Disk", "Memory"
  ];

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

  $scope.loadChart = function(device, chart) {
    console.log(device.ip, chart);
    var templDef = templateDefinitions[chart];
    templDef.source = device.ip;

    console.log(templDef);

    device.charts = device.charts || [];
    device.charts.push(templDef);
  };
}

function FlowsCtrl($scope, $http) {
  $http.get(flowURL).then(function(response) {
    if (response.status == 200) {
      $scope.flows = response.data.data;     
    }
  });
}
