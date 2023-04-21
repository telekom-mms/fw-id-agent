package agent

import (
	"reflect"
	"testing"

	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
)

// TestParseTNDServers tests parseTNDServers
func TestParseTNDServers(t *testing.T) {
	// test invalid
	got, ok := parseTNDServers("")
	if ok {
		t.Errorf("got true, want false")
	}

	// test single valid
	want := []config.TNDHTTPSConfig{
		{
			URL:  "https://testserver1.com:8443",
			Hash: "abcdef1234567890",
		},
	}
	got, ok = parseTNDServers(want[0].URL + ":" + want[0].Hash)
	if !ok {
		t.Errorf("got false, want true")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test multiple valid
	want = []config.TNDHTTPSConfig{
		{
			URL:  "https://testserver1.com:8443",
			Hash: "abcdef1234567890",
		},
		{
			URL:  "https://testserver2.com",
			Hash: "abcdef1234567890",
		},
		{
			URL:  "https://192.168.1.1:9443",
			Hash: "abcdef1234567890",
		},
		{
			URL:  "https://192.168.2.1",
			Hash: "abcdef1234567890",
		},
	}
	got, ok = parseTNDServers(want[0].URL + ":" + want[0].Hash + "," +
		want[1].URL + ":" + want[1].Hash + "," +
		want[2].URL + ":" + want[2].Hash + "," +
		want[3].URL + ":" + want[3].Hash)
	if !ok {
		t.Errorf("got false, want true")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
