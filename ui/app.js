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

cisternApp.directive('uiChart', function() {
  return {
    restrict: 'AE',
    replace: 'true',
    template: '<span></span>',
    link: function(scope, el, attr) {
      scope.source = attr.source;
      scope.desc = attr.desc;

      try {
        scope.metrics = JSON.parse(attr.metrics);
      } catch(axis) {
        scope.metrics = [];
      }

      try {
        scope.factors = JSON.parse(attr.factors);
      } catch(a) {
        scope.factors = null;
      }

      var generateC3Data = function(template, series) {
        var columns = {};
        var xs = {};
        var types = {};
        var groups = [template.metrics];

        for(var i in template.metrics) {
          types[template.metrics[i]] = 'area-step';
        }

        for(var i in series) {
          var s = series[i];
          var xColName = s.metric + '.x';
          var xCol = columns[xColName] || [];
          var yCol = columns[s.metric] || [];

          var factor = 1;
          if(template.factors) {
            var factorIndex = 0;
            for(var j in template.metrics) {
              if(template.metrics[j] == s.metric) {
                factorIndex = j;
              }
            }

            factor = template.factors[factorIndex];
          }

          var points = s['points'];
          for(var j in points) {
            xCol.push(new Date(1000*points[j].timestamp));
            yCol.push(points[j].value * factor);
          }

          columns[xColName] = xCol;
          columns[s.metric] = yCol;

          xs[s.metric] = xColName;
        }

        var cols2 = columns;
        columns = [];
        for(var colname in cols2) {
          var col = cols2[colname];
          col.unshift(colname);
          columns.push(col);
        }

        return {
          columns: columns,
          xs: xs,
          types: types,
          groups: groups
        };
      };

      var plotStackedMetrics = function(template, c3Data, elem) {
        var metricDiv = document.createElement('div');
        metricDiv.className = 'plot-metric';
        var plotDiv = document.createElement('div');
        plotDiv.className = 'plot';

        metricDiv.innerHTML = "<span class='chart-desc'>" + template.desc + "</span>";
        metricDiv.appendChild(plotDiv);
        elem.append(metricDiv);

        c3.generate({
          bindto: plotDiv,
          data: {
            xs: c3Data.xs,
            columns: c3Data.columns,
            types: c3Data.types,
            groups: c3Data.groups,
            order: null
          },
          axis: {
            x: {
              type: 'timeseries',
              tick: {
                format:'%I:%M:%S',
                count: 3
              }
            },
            y: {
              tick: {
                count: 5,
                format: d3.format('.1s')
              },
              center: 0
            }
          },
          legend: {
            show: false
          },
          point: {
            show: false
          }
        });
      }

      var timePeriod = 86400;

      var rows = [];
      for(var j in scope.metrics) {
        rows.push({source: scope.source, metric: scope.metrics[j], start: -timePeriod, end: -1});
      }

      var closure = function(template, elem) {
        return function(data, status) {
          var series = JSON.parse(data).data.series;
          if (!series) {
            return;
          }

          plotStackedMetrics(template, generateC3Data(template, JSON.parse(data).data.series), elem)
        };
      };

      $.ajax({
        url: cisternURL+'/series/query?pointWidth=' + 240,
        data: JSON.stringify(rows),
        success: closure({source: scope.source, desc: scope.desc, metrics: scope.metrics, factors: scope.factors}, el),
        contentType: "application/json",
        method: "POST"
      });
    }
  };
});
