package message

type Message struct {
	Global  bool        // A message in the global scope. False corresponds to a device scope.
	Class   string      // Class name
	Type    string      // Type within class
	Content interface{} // Arbitrary content
}
