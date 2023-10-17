package krbmon

import "testing"

// TestConfMonStartStop tests starting and stopping of ConfMon
func TestConfMonStartStop(_ *testing.T) {
	cm := NewConfMon()
	cm.Start()
	cm.Stop()
}
