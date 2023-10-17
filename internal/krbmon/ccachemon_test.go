package krbmon

import "testing"

// TestCCacheStartStop tests starting and stopping of CCacheMon
func TestCCacheMonStartStop(_ *testing.T) {
	cm := NewCCacheMon()
	cm.Start()
	cm.Stop()
}
