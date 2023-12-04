package krbmon

import "github.com/fsnotify/fsnotify"

// fsnotifyNewWatcher is fsnotify.NewWatcher for testing.
var fsnotifyNewWatcher = fsnotify.NewWatcher

// watcherAdd is watcher.Add for testing.
var watcherAdd = func(watcher *fsnotify.Watcher, name string) error {
	return watcher.Add(name)
}

// watcherClose is watcher.Close for testing.
var watcherClose = func(watcher *fsnotify.Watcher) error {
	return watcher.Close()
}
