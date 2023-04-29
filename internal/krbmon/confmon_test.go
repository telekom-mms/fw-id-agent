package krbmon

import "testing"

// TestConfMonStartStop tests starting and stopping of ConfMon
func TestConfMonStartStop(t *testing.T) {
	cm := NewConfMon()
	cm.Start()
	cm.Stop()
}
