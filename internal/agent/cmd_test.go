package agent

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/fw-id-agent/pkg/config"
)

// TestParseTNDServers tests parseTNDServers.
func TestParseTNDServers(t *testing.T) {
	// test invalid, empty
	_, ok := parseTNDServers("")
	if ok {
		t.Errorf("got true, want false")
	}

	// test invalid, wrong format
	_, ok = parseTNDServers("example.com:")
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
	got, ok := parseTNDServers(want[0].URL + ":" + want[0].Hash)
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

// TestGetConfig tests getConfig.
func TestGetConfig(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		dir := t.TempDir()
		conf := filepath.Join(dir, "does-not-exist")
		args := []string{"test", fmt.Sprintf("--%s=%s", argConfig, conf)}
		_, err := getConfig(args)
		if err == nil {
			t.Error("no config should fail")
		}
	})

	t.Run("invalid argument", func(t *testing.T) {
		args := []string{"test", "-this-argument-does-not-exist"}
		_, err := getConfig(args)
		if err == nil {
			t.Error("invalid argument should fail")
		}
	})

	t.Run("help", func(t *testing.T) {
		args := []string{"test", "-help"}
		_, err := getConfig(args)
		if err != flag.ErrHelp {
			t.Errorf("help should return flag.ErrHelp: err %v", err)
		}
	})

	t.Run("version", func(t *testing.T) {
		args := []string{"test", fmt.Sprintf("--%s", argVersion)}
		_, err := getConfig(args)
		if err != flag.ErrHelp {
			t.Errorf("version should return flag.ErrHelp: err %v", err)
		}
	})

	t.Run("invalid TND servers", func(t *testing.T) {
		args := []string{"test", fmt.Sprintf("--%s=invalid", argTNDServers)}
		_, err := getConfig(args)
		if err == nil {
			t.Error("invalid TND servers should fail")
		}
	})

	t.Run("invalid serviceURL", func(t *testing.T) {
		args := []string{"test", fmt.Sprintf("--%s=\"\"", argServiceURL)}
		_, err := getConfig(args)
		if err == nil {
			t.Error("invalid serviceURL should fail")
		}
	})

	t.Run("valid", func(t *testing.T) {
		dir := t.TempDir()
		conf := filepath.Join(dir, "exists")
		if err := os.WriteFile(conf, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		args := []string{"test",
			fmt.Sprintf("--%s=%s", argConfig, conf),
			fmt.Sprintf("--%s=example", argServiceURL),
			fmt.Sprintf("--%s=example", argRealm),
			fmt.Sprintf("--%s=300", argKeepAlive),
			fmt.Sprintf("--%s=5", argLoginTimeout),
			fmt.Sprintf("--%s=2", argLogoutTimeout),
			fmt.Sprintf("--%s=1", argRetryTimer),
			fmt.Sprintf("--%s=example:abcdef", argTNDServers),
			fmt.Sprintf("--%s=false", argVerbose),
			fmt.Sprintf("--%s=0", argStartDelay),
			fmt.Sprintf("--%s=false", argNotifications),
		}

		cfg, err := getConfig(args)
		if cfg == nil || err != nil {
			t.Errorf("should get valid config: cfg %v, err %v", cfg, err)
		}
	})
}

// TestSetVerbose tests setVerbose.
func TestSetVerbose(t *testing.T) {
	// test normal output
	cfg := config.Default()
	setVerbose(cfg)
	if log.GetLevel() != log.InfoLevel {
		t.Error("log level should be info")
	}

	// test verbose output
	cfg.Verbose = true
	setVerbose(cfg)
	if log.GetLevel() != log.DebugLevel {
		t.Error("log level should be debug")
	}
}
