package krbmon

import "testing"

// TestCCacheStartStop tests starting and stopping of CCacheMon
func TestCCacheMonStartStop(t *testing.T) {
	cm := NewCCacheMon()
	cm.Start()
	cm.Stop()
}
