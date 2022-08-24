package notify

import "testing"

// TestNotify tests Notify
func TestNotify(t *testing.T) {
	Notify("test", "this is a test")
}
