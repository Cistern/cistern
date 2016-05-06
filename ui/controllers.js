var cisternURL = 'http://localhost:8080'
var flowURL = 'http://localhost:8080/devices/127.0.0.1/flows'

var templateDefinitions = {
  "CPU":
  {
    "desc": "CPU usage",
    "metrics": ['cpu.user', 'cpu.sys', 'cpu.nice', 'cpu.softintr', 'cpu.intr', 'cpu.wio', 'cpu.idle']
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
    "metrics": ['mem.used', 'mem.shared', 'mem.buffers', 'mem.cached', 'mem.free']
  },
  
  "Network":
  {
    "desc": "Network traffic",
    "metrics": ['net.bytes_out', 'net.bytes_in'],
    "factors": [-1, 1]
  }
};

function NavigationCtrl($scope, $location) {
  $scope.isActive = function(viewLocation) {
    return viewLocation === $location.path();
  }
}

function DevicesCtrl($scope, $http) {
  $scope.charts = [
    "CPU", "Disk", "Memory", "Network"
  ];

  $http.get(cisternURL+'/sources/').then(function(response) {
    if (response.status == 200) {
      $scope.devices = [];
      for (var i = 0; i < response.data.data.length; i++) {
        $scope.devices.push({
          name: response.data.data[i]
        })
      }

      $scope.devices.sort(function(a, b) {
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

  $scope.loadChart = function(device, chart) {
    console.log(device, chart);
    var templDef = templateDefinitions[chart];
    templDef.source = device.name;

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
