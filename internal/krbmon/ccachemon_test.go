package krbmon

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/test/testdata"
	log "github.com/sirupsen/logrus"
)

// TestCCacheUpdateGetTGT tests GetTGT of CCacheUpdate.
func TestCCacheUpdateGetTGT(t *testing.T) {
	// prepare test ccache update
	b, err := hex.DecodeString(testdata.CCACHE_TEST)
	if err != nil {
		t.Fatal(err)
	}
	ccache := new(credentials.CCache)
	err = ccache.Unmarshal(b)
	if err != nil {
		t.Fatal(err)
	}
	c := &CCacheUpdate{ccache}

	// test wrong realm
	if c.GetTGT("invalid") != nil {
		t.Error("should not return TGT for wrong realm")
	}

	// test correct realm
	if c.GetTGT("TEST.GOKRB5") == nil {
		t.Error("should return TGT for correct realm")
	}
}

// TestGetCredentialCacheFilename tests getCredentialCacheFilename.
func TestGetCredentialCacheFilename(t *testing.T) {
	defer func() { userCurrent = user.Current }()

	// test without error
	userCurrent = func() (*user.User, error) {
		return &user.User{Uid: "1", Gid: "1", Username: "user", Name: "user", HomeDir: "/home/user"}, nil
	}

	// variable not set
	if err := os.Setenv("KRB5CCNAME", ""); err != nil {
		t.Fatal(err)
	}
	if f, err := getCredentialCacheFilename(); err != nil {
		t.Error(err)
	} else if f != "/tmp/krb5cc_1" {
		t.Errorf("wrong ccache filename %s", f)
	}

	// variable set, but wrong
	if err := os.Setenv("KRB5CCNAME", "invalid"); err != nil {
		t.Fatal(err)
	}
	if f, err := getCredentialCacheFilename(); err != nil {
		t.Error(err)
	} else if f != "/tmp/krb5cc_1" {
		t.Errorf("wrong ccache filename %s", f)
	}

	// variable set
	if err := os.Setenv("KRB5CCNAME", "FILE:/test/file"); err != nil {
		t.Fatal(err)
	}
	if f, err := getCredentialCacheFilename(); err != nil {
		t.Error(err)
	} else if f != "/test/file" {
		t.Errorf("wrong ccache filename %s", f)
	}

	// test with error
	userCurrent = func() (*user.User, error) {
		return nil, errors.New("test error")
	}

	if err := os.Setenv("KRB5CCNAME", ""); err != nil {
		t.Fatal(err)
	}
	if f, err := (getCredentialCacheFilename()); f != "" && err == nil {
		t.Error("file should be empty string and error should not be nil")
	}
}

// TestCCacheMonIsCCacheFileEvent tests isCCacheFileEvent of CCacheMon.
func TestCCacheMonIsCCacheFileEvent(t *testing.T) {
	c := NewCCacheMon()
	c.cCacheFile = "/test/file"

	e := fsnotify.Event{}

	// test no ccache event
	e.Name = "wrong"
	if c.isCCacheFileEvent(e) {
		t.Errorf("%v should not be a ccache file event", e)
	}

	// test ccache event
	e.Name = "/test/file"
	if !c.isCCacheFileEvent(e) {
		t.Errorf("%v should be a ccache file event", e)
	}
}

// TestCCacheHandleCCacheFileEvent tests handleCCacheFileEvent of CCacheMon.
func TestCCacheHandleCCacheFileEvent(t *testing.T) {
	// create directory and ccache file name
	dir := t.TempDir()
	c := NewCCacheMon()
	c.cCacheFile = filepath.Join(dir, "ccache")

	go func() {
		defer close(c.updates)

		// handle unrelated file -> no update
		c.handleCCacheFileEvent(fsnotify.Event{Name: "other"})

		// handle not existing ccache file -> no update
		c.handleCCacheFileEvent(fsnotify.Event{Name: c.cCacheFile})

		// handle empty ccache file -> no update
		if err := os.WriteFile(c.cCacheFile, []byte(""), 0666); err != nil {
			panic(err)
		}
		c.handleCCacheFileEvent(fsnotify.Event{Name: c.cCacheFile})

		// handle invalid ccache -> no update
		garbage := [512]byte{}
		if err := os.WriteFile(c.cCacheFile, garbage[:], 0666); err != nil {
			panic(err)
		}
		c.handleCCacheFileEvent(fsnotify.Event{Name: c.cCacheFile})

		// handle valid ccache -> update
		b, err := hex.DecodeString(testdata.CCACHE_TEST)
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(c.cCacheFile, b, 0666); err != nil {
			panic(err)
		}
		c.handleCCacheFileEvent(fsnotify.Event{Name: c.cCacheFile})

		// handle valid ccache again with no changes -> no update
		c.handleCCacheFileEvent(fsnotify.Event{Name: c.cCacheFile})
	}()

	// collect and count updates
	want := 1
	got := 0
	for range c.Updates() {
		got++
	}
	if got != want {
		t.Errorf("unexpected number of updates: got %d, want %d", got, want)
	}
}

// TestCCacheHandleCCacheFileError tests handleCCacheFileError of CCacheMon.
func TestCCacheHandleCCacheFileError(t *testing.T) {
	// log to buffer
	oldOut := log.StandardLogger().Out
	b := &bytes.Buffer{}
	log.SetOutput(b)
	defer func() { log.SetOutput(oldOut) }()

	// handle error and check log
	c := NewCCacheMon()
	c.handleCCacheFileError(errors.New("test error"))
	got := b.String()
	if got == "" {
		t.Error("empty log")
	}
}

// TestCCacheStartStop tests starting and stopping of CCacheMon.
func TestCCacheMonStartStop(t *testing.T) {
	t.Run("start and stop without errors", func(t *testing.T) {
		// test without errors
		cm := NewCCacheMon()
		if err := cm.Start(); err != nil {
			t.Errorf("could not start monitor: %v", err)
		}
		cm.Stop()
	})

	t.Run("start and handle events", func(t *testing.T) {
		// create dummy monitor with not existing ccache file
		dir := t.TempDir()
		cm := NewCCacheMon()
		cm.cCacheDir = dir
		cm.cCacheFile = filepath.Join(dir, "does-not-exist")
		watcher, _ := fsnotify.NewWatcher()
		cm.watcher = watcher

		go func() {
			// start monitor
			go cm.start()

			// send unrelated event to trigger event handler
			watcher.Events <- fsnotify.Event{}

			// send error to trigger error handler
			watcher.Errors <- errors.New("test error")

			// close watcher to trigger closing of channels and monitor exit
			if err := watcher.Close(); err != nil {
				panic(err)
			}
		}()

		// collect and count updates
		want := 0
		got := 0
		for range cm.Updates() {
			got++
		}
		if got != want {
			t.Errorf("unexpected number of updates: got %d, want %d", got, want)
		}
	})

	t.Run("start with errors", func(t *testing.T) {
		// test with Watcher.Add() error
		oldWatcherAdd := watcherAdd
		watcherAdd = func(watcher *fsnotify.Watcher, name string) error {
			return errors.New("test error")
		}
		defer func() { watcherAdd = oldWatcherAdd }()
		cm := NewCCacheMon()
		if err := cm.Start(); err == nil {
			t.Error("monitor should not start")
		}

		// test with NewWatcher() error
		fsnotifyNewWatcher = func() (*fsnotify.Watcher, error) {
			return nil, errors.New("test error")
		}
		defer func() { fsnotifyNewWatcher = fsnotify.NewWatcher }()
		cm = NewCCacheMon()
		if err := cm.Start(); err == nil {
			t.Error("monitor should not start")
		}

		// test with credential cache file name error
		oldGetCCacheFile := getCredentialCacheFilename
		getCredentialCacheFilename = func() (string, error) {
			return "", errors.New("test error")
		}
		defer func() { getCredentialCacheFilename = oldGetCCacheFile }()
		cm = NewCCacheMon()
		if err := cm.Start(); err == nil {
			t.Error("monitor should not start")
		}
	})

	t.Run("stop with error", func(t *testing.T) {
		// set watcher close function that returns error
		oldWatcherClose := watcherClose
		watcherClose = func(watcher *fsnotify.Watcher) error {
			_ = watcher.Close()
			return errors.New("test error")
		}
		defer func() { watcherClose = oldWatcherClose }()

		// create monitor with not existing file
		dir := t.TempDir()
		watcher, _ := fsnotify.NewWatcher()
		cm := NewCCacheMon()
		cm.cCacheDir = dir
		cm.cCacheFile = filepath.Join(dir, "does-not-exist")
		cm.watcher = watcher

		// start and stop monitor
		go cm.start()
		cm.Stop()
		<-cm.Updates()
	})
}

// TestCCacheMonUpdates tests Updates of CCacheMon.
func TestCCacheMonUpdates(t *testing.T) {
	c := NewCCacheMon()
	if c.Updates() != c.updates {
		t.Errorf("invalid updates channel in %v", c)
	}
}

// TestNewCCacheMon tests NewCCacheMon.
func TestNewCCacheMon(t *testing.T) {
	c := NewCCacheMon()
	if c == nil || c.updates == nil || c.done == nil {
		t.Errorf("invalid ccachemon %v", c)
	}
}
