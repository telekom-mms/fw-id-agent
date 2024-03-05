package krbmon

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/jcmturner/gokrb5/v8/test/testdata"
	log "github.com/sirupsen/logrus"
)

// TestConfMonIsConfigFileEvent tests isConfigFileEvent of ConfMon.
func TestConfMonIsConfigFileEvent(t *testing.T) {
	c := NewConfMon()
	e := fsnotify.Event{}

	// test no file event
	e.Name = "/other/file"
	if c.isConfigFileEvent(e) {
		t.Errorf("%v should not be file event", e)
	}

	// test file event
	e.Name = "/etc/krb5.conf"
	if !c.isConfigFileEvent(e) {
		t.Errorf("%v should be file event", e)
	}
}

// TestConfMonHandleConfigFileEvent tests handleConfigFileEvent of ConfMon.
func TestConfMonHandleConfigFileEvent(t *testing.T) {
	// create directory and config file name
	dir := t.TempDir()
	c := NewConfMon()
	c.confDir = filepath.Dir(dir)
	c.confFile = filepath.Join(dir, "conf")

	go func() {
		defer close(c.updates)

		// handle unrelated event -> no update
		c.handleConfigFileEvent(fsnotify.Event{Name: "other"})

		// handle not existing config file -> no update
		c.handleConfigFileEvent(fsnotify.Event{Name: c.confFile})

		// handle empty config file -> update
		if err := os.WriteFile(c.confFile, []byte(""), 0666); err != nil {
			panic(err)
		}
		c.handleConfigFileEvent(fsnotify.Event{Name: c.confFile})

		// handle invalid config file -> no update
		garbage := [512]byte{}
		if err := os.WriteFile(c.confFile, garbage[:], 0666); err != nil {
			panic(err)
		}
		c.handleConfigFileEvent(fsnotify.Event{Name: c.confFile})

		// handle valid config file -> update
		if err := os.WriteFile(c.confFile, []byte(testdata.KRB5_CONF), 0666); err != nil {
			panic(err)
		}
		c.handleConfigFileEvent(fsnotify.Event{Name: c.confFile})

		// handle valid config file again with no changes -> no update
		c.handleConfigFileEvent(fsnotify.Event{Name: c.confFile})
	}()

	// collect and count updates
	want := 2
	got := 0
	for range c.Updates() {
		got++
	}
	if got != want {
		t.Errorf("unexpected number of updates: got %d, want %d", got, want)
	}
}

// TestConfMonHandleConfigFileError tests handleConfigFileError of ConfMon.
func TestConfMonHandleConfigFileError(t *testing.T) {
	// log to buffer
	oldOut := log.StandardLogger().Out
	b := &bytes.Buffer{}
	log.SetOutput(b)
	defer func() { log.SetOutput(oldOut) }()

	// handle error and check log
	cm := NewConfMon()
	cm.handleConfigFileError(errors.New("test error"))
	got := b.String()
	if got == "" {
		t.Error("empty log")
	}
}

// TestConfMonStartStop tests starting and stopping of ConfMon.
func TestConfMonStartStop(t *testing.T) {
	t.Run("start and stop without errors", func(t *testing.T) {
		// test without errors
		cm := NewConfMon()
		if err := cm.Start(); err != nil {
			t.Errorf("could not start monitor: %v", err)
		}
		cm.Stop()
	})

	t.Run("start and handle events", func(t *testing.T) {
		// create dummy monitor with not existing config file
		dir := t.TempDir()
		watcher, _ := fsnotify.NewWatcher()
		cm := NewConfMon()
		cm.confDir = dir
		cm.confFile = filepath.Join(dir, "does-not-exist")
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
		cm := NewConfMon()
		if err := cm.Start(); err == nil {
			t.Error("monitor should not start")
		}

		// test with NewWatcher() error
		fsnotifyNewWatcher = func() (*fsnotify.Watcher, error) {
			return nil, errors.New("test error")
		}
		defer func() { fsnotifyNewWatcher = fsnotify.NewWatcher }()
		cm = NewConfMon()
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
		cm := NewConfMon()
		cm.confDir = dir
		cm.confFile = filepath.Join(dir, "does-not-exist")
		cm.watcher = watcher

		// start and stop monitor
		go cm.start()
		cm.Stop()
		<-cm.Updates()
	})
}

// TestConfMonUpdates tests Updates of ConfMon.
func TestConfMonUpdates(t *testing.T) {
	c := NewConfMon()
	if c.Updates() != c.updates {
		t.Error("invalid updates")
	}
}

// TestNewConfMon tests NewConfMon.
func TestNewConfMon(t *testing.T) {
	c := NewConfMon()
	if c == nil ||
		c.confFile == "" ||
		c.confDir == "" ||
		c.updates == nil ||
		c.done == nil ||
		c.closed == nil {

		t.Error("invalid ConfMon")
	}
}
