var m = require("mithril");
var d3 = require("d3");
var groupColor = require("./groupColor");

var Chart = function(chartState) {
  this.oninit = function(vnode) {
    vnode.state.chartState = chartState;
  };
  this.view = function(vnode) {
    // Resize
    var resize = function(vnode) {
      var chart = vnode.dom;
      var width = parseInt(d3.select(chart).style("width"));
      var data = this.chartState.data,
               w = width,
               h = this.chartState.height,
               margin = 35,
               y = d3.scaleLinear().domain([ 0, this.chartState.maxVal * 1.1 ]).range([ h - margin, 0 ]),
               x = d3.scaleTime().domain([ this.chartState.start, this.chartState.end ]).range([ 0 + 2*margin, w - margin ]);
      var yAxis = d3.axisLeft(y).ticks(3).tickFormat(d3.format(".0s"));
      var xAxis = d3.axisBottom(x).ticks(4);
      // Remove existing paths
      d3.select(chart).selectAll("path").remove();

      // Remove tooltip box.
      d3.select(chart).selectAll("tooltip").remove();

      var tooltipData = [];

      // Draw paths
      for (i in data.lines) {
        var lineData = data.lines[i];
        var line = d3.line().x(function(d, i) {
          return x(d.ts);
        }).y(function(d) {
          return y(d.y);
        });
        tooltipData.push({"name": i, "data": data.lines[i], "line": line})
        var color = groupColor(i);
        d3.select(chart).select(".lineGroup").append("path")
          .attr("d", line(lineData))
          .attr("fill", "none")
          .attr("stroke", color)
          .attr("stroke-width", "1px")
          .attr("class", i);
      }
      // Draw axes
      d3.select(chart).select(".y-axis").attr("transform", "translate(" + (2*margin-10) + ", 0)").call(yAxis);
      d3.select(chart).select(".x-axis").attr("transform", "translate(0, " + (h - margin + 10) + ")").call(xAxis);

      var tooltipLine = d3.select(chart).select(".lineGroup").append("line");

      var drawTooltip = function() {
        var hoverDate = new Date(x.invert(d3.mouse(d3.select(chart).select(".overlay").node())[0]));
        tooltipLine.attr('stroke', 'black')
          .attr('x1', x(hoverDate))
          .attr('x2', x(hoverDate))
          .attr('y1', 0)
          .attr('y2', h - margin);

          var getValue = function(d) {
            var y1 = 0;
            var y2 = 0;
            var x1;
            var x2;
            var value = 0;
            for (i in d.data) {
              if (d.data[i].ts > hoverDate) {
                x2 = d.data[i].ts;
                y2 = d.data[i].y;
                break;
              }
              x1 = d.data[i].ts;
              y1 = d.data[i].y;
            }
            var percent = (hoverDate-x1)/(x2-x1);
            value = y1 + (percent*(y2-y1))
            if (value != value) {
              return 0;
            }
            return value;
          }

          // sort tooltip data
          tooltipData.sort(function(a, b) {
            return getValue(b) - getValue(a);
          });

          console.log(d3.event.pageX, d3.event.pageY)

          var tooltip = d3.select(document).select(".chart-tooltip");
          tooltip.html(hoverDate)
            .style("position", "absolute")
            .style('display', 'block')
            .style('left', d3.event.pageX+10 + "px")
            .style('top', d3.event.pageY + "px")
            .selectAll()
            .data(tooltipData)
            .enter()
            .append("div")
            .html(function(d) {
              var value = getValue(d);
              if (value == 0) {
                return;
              }
              return "<div style='color: "+ groupColor(d.name) + "'>" +
                "<strong>"+d.name + "</strong>: " + d3.format(".4s")(value) +
                "</div>";
            })
      }

      var removeTooltip = function() {
        d3.select(document).select('.chart-tooltip').style("display", "none");
        tooltipLine.attr("stroke", "none");
      }

      // Set up brush
      brushended = function() {
        var s = d3.event.selection;
        if (s) {
            var start = x.invert(s[0]);
            var end = x.invert(s[1]);
            this.chartState.brushEnd(start, end)
        } else {
            var end = new Date();
            var start = new Date(end - 90*86400*1000);
            this.chartState.brushEnd(start, end)
        }
      }.bind(this);
      var brush = d3.brushX().on("end", brushended).extent([ [ margin, 0 ], [ w - margin, h - margin ] ]);
      d3.select(chart).select(".brush").call(brush);
      d3.select(chart).select(".overlay")
        .on('mousemove', drawTooltip)
        .on('mouseout', removeTooltip);
    }.bind(this);
    // Draw
    var draw = function(vnode) {
      console.log("this.chartState.name = ", this.chartState.name);
      d3.select(window).on("resize." + this.chartState.name, resize.bind(null, vnode));
      resize(vnode);
    }.bind(this);
    // Elements
    return m("div.chart", [
      m("h4.chart-name", this.chartState.name),
      m("svg", {
        width: "100%",
        height: this.chartState.height,
        oncreate: draw.bind(this)
      },
      m("g", [ m("g.lineGroup"), m("g.x-axis"), m("g.y-axis"), m("g.brush") ]))
    ]);
  };
};

module.exports = Chart;
