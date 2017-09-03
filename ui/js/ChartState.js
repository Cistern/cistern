var ChartState = function(width, height, start, end, data, name, brushEnd) {
  this.width = width;
  this.height = height;
  this.start = start;
  this.end = end;
  this.data = data;
  this.name = name;
  this.maxVal = 10;
  this.brushEnd = brushEnd
  for (var i in data.lines) {
    var lineData = data.lines[i];
    if (lineData.length == 1) {
      // Only one point, so skip it.
      continue
    }
    for (var j in lineData) {
      if (lineData[j].y > this.maxVal) {
        this.maxVal = lineData[j].y;
      }
    }
  }
};

module.exports = ChartState;
