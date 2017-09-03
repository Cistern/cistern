var m = require("mithril");
var ChartContainer = require("./ChartContainer")

var App = {
  view: function(vnode) {
    return m("div", "todo");
  }
};

var CollectionChartPage = {
  view: function(vnode) {
    return m("div", [ m("div", m(ChartContainer)) ]);
  }
};

m.mount(document.getElementById("app"), CollectionChartPage);
