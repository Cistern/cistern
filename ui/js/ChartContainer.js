var m = require("mithril");
var d3 = require("d3");
var ChartState = require("./ChartState");
var Chart = require("./Chart");
var groupColor = require("./groupColor");

var ChartContainer = {
  oninit: function(vnode) {

    var queryString = m.parseQueryString(window.location.search.replace(/^[?]/, ''))
    var start = new Date(), end = new Date();
    if (queryString.start) {
      start = new Date(queryString.start)
    }
    if (queryString.end) {
      end = new Date(queryString.end)
    }

    var data = {
      series: [],
      query: {
        time_range: {
          start: new Date(),
          end: new Date()
        }
      }
    };
    vnode.state.start = start;
    vnode.state.end = end;
    vnode.state.query = queryString.query || "";
    vnode.state.collection = queryString.collection || "";

    vnode.state.brushEnd = function(start, end) {
      this.start = start
      this.end = end
      this.refresh()
      this.updateURL()
    }.bind(vnode.state)

    vnode.state.refresh = function() {
      var process = function(data) {
        var columnToName = function(column) {
          return column.aggregate + "(" + column.name + ")";
        };
        var charts = [];
        var chartData = {};
        if (data.query.point_size > 0) {
          for (var i in data.query.columns) {
            var column = data.query.columns[i];
            var name = columnToName(column);
            charts.push(name);
            chartData[name] = {};
          }
        }
        for (var i in data.series) {
          var point = data.series[i];
          var groupID = point["_group_id"];
          for (var j in charts) {
            var chartName = charts[j];
            if (!chartData[chartName][groupID]) {
              chartData[chartName][groupID] = [];
            }
            chartData[chartName][groupID].push({
              ts: new Date(point["_ts"]),
              y: point[chartName]
            });
          }
        }

        this.chartStates = {};
        for (var i in chartData) {
          this.chartStates[i] = new ChartState(
            100,
            200,
            new Date(data.query.time_range.start),
            new Date(data.query.time_range.end),
            { lines: chartData[i] },
            i,
            this.brushEnd
          );
        }

        this.summaryRows = data.summary;
        this.events = data.events;
        this.start = new Date(data.query.time_range.start);
        this.end = new Date(data.query.time_range.end);

      }.bind(this)

      vnode.state.updateURL = function() {
        var queryString = m.parseQueryString(window.location.search.replace(/^[?]/, ''))
        queryString.start = this.start.toJSON()
        queryString.end = this.end.toJSON()
        queryString.query = this.query
        queryString.collection = this.collection

        var newurl = window.location.protocol + "//" + window.location.host + window.location.pathname + '?' + m.buildQueryString(queryString);
        window.history.pushState({path: newurl}, '', newurl);
      }.bind(this)

      m.request({
        method: "POST",
        url: "/api/collections/" + this.collection + "/query?" +
          "start=" + Math.floor(this.start.getTime()/1000) + "&" +
          "end=" + Math.floor(this.end.getTime()/1000) + "&" +
          "query=" + encodeURIComponent(this.query)
      }).then(process);
    }.bind(vnode.state);

    window.onpopstate = (function(e) {
      this.refresh()
    }).bind(vnode.state);

    vnode.state.refresh()
  },
  view: function(vnode) {
    var chartComponents = [];
    console.log(vnode.state.chartStates)
    for (var i in vnode.state.chartStates) {
      chartComponents.push(m(new Chart(vnode.state.chartStates[i])));
    }

    var summaryRows = vnode.state.summaryRows;
    var summaryTable;
    if (summaryRows) {
      var headers = Object.keys(summaryRows[0])
      summaryTable = m("table.pure-table", [
        m("thead",
          m("tr", headers.map(function(d) {
            if (d == "_group_id") {
              return m("th", "Group")
            }
            return m("th", d)
          }))
        ),
        m("tbody", summaryRows.map(function(row) {
          return m("tr", Object.keys(row).map(function(k) {
            if (k == "_group_id") {
              return m("td", m("div",
                {
                  style: {
                    color: groupColor(row[k])
                  }
                },
                row[k]))
            }
            return m("td", row[k])
          }))
        }))
      ])
    }

    var events = vnode.state.events;
    var eventsTable;
    if (events) {
      var headers = Object.keys(events[0])
      eventsTable = m("table.pure-table", [
        m("thead",
          m("tr", headers.map(function(d) {
            if (d == "_id") {return}
            return m("th", d)
          }))
        ),
        m("tbody", events.map(function(row) {
          return m("tr", Object.keys(row).map(function(k) {
            if (k == "_id") {return}
            return m("td", row[k])
          }))
        }))
      ])
    }

    var resultsComponents = [];

    if (chartComponents.length > 0) {
      resultsComponents.push(m("div.row", [
        m("h2", "Series"),
        chartComponents
      ]));
    }

    if (summaryRows) {
      resultsComponents.push(m("div", {className: "row summary-table"}, [
        m("h2", "Summary"),
        summaryTable
      ]));
    }

    if (events) {
      resultsComponents.push(m("div", {className: "row events-table"}, [
        m("h2", "Events"),
        eventsTable
      ]));
    }

    var collectionInputField = m("input.form-control", {
      onchange: m.withAttr("value", function(v) {
        vnode.state.collection = v;
        vnode.state.refresh();
        vnode.state.updateURL();
      }),
      size: 30,
      id: "query-collection",
      value: vnode.state.collection
    })

    var startInputField = m("input.form-control", {
      onchange: m.withAttr("value", function(v) {
        var d = new Date(v);
        if (!isNaN(d.getTime())) {
          vnode.state.start = new Date(v);
          vnode.state.refresh();
          vnode.state.updateURL();
          return;
          for (var i in vnode.state.chartStates) {
            vnode.state.chartStates[i].start = new Date(v);
          }
        }
      }),
      size: 30,
      id: "query-start",
      value: vnode.state.start.toJSON()
    });

    var endInputField = m("input.form-control", {
      onchange: m.withAttr("value", function(v) {
        var d = new Date(v);
        if (!isNaN(d.getTime())) {
          vnode.state.end = new Date(v);
          vnode.state.refresh();
          vnode.state.updateURL();
          return;
          for (var i in vnode.state.chartStates) {
            vnode.state.chartStates[i].end = new Date(v);
          }
        }
      }),
      size: 30,
      id: "query-end",
      value: vnode.state.end.toJSON()
    });

    var queryField = m("textarea", {
      onchange: m.withAttr("value", function(v) {
        vnode.state.query = v;
        vnode.state.refresh();
        vnode.state.updateURL();
      }),
      style: { width: "100%" },
      id: "query-text",
      value: vnode.state.query
    });


    var inputs = m("form", {
      "class": "pure-form pure-form-stacked"
    }, [
      m("fieldset", [
        m("div.pure-g", [
          m("div.pure-u-1-3", [
            m("label", {for: "query-collection"}, "Collection"),
            collectionInputField
          ]),
          m("div.pure-u-1-3", [
            m("label", {for: "query-start"}, "Start timestamp"),
            startInputField
          ]),
          m("div.pure-u-1-3", [
            m("label", {for: "query-end"}, "End timestamp"),
            endInputField
          ]),
        ]),
        m("div.pure-g", [
          m("div.pure-u-1", [
            m("label", {for: "query-text"}, "Query"),
            queryField
          ])
        ])
      ])
    ])

    return m("div", [
      inputs,

      m("div", resultsComponents)
    ]);
  }
};

module.exports = ChartContainer;
