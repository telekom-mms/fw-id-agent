package notify

import "testing"

// TestNotify tests Notify
func TestNotify(t *testing.T) {
	n := NewNotifier()
	n.Notify("test", "this is a test")
	n.Close()
}
