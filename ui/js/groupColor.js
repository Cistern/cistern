var groupColor = function(groupID) {
  var hue = parseInt(groupID, 16)
  hue = hue % 360
  return "hsl(" + hue + ", 50%, 50%)"
}

module.exports = groupColor;
