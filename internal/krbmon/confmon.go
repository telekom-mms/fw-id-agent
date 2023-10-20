package krbmon

import (
	"path/filepath"
	"reflect"

	"github.com/fsnotify/fsnotify"
	"github.com/jcmturner/gokrb5/v8/config"
	log "github.com/sirupsen/logrus"
)

var (
	// krb5conf is the file path of the krb5.conf file.
	krb5conf = "/etc/krb5.conf"
)

// ConfUpdate is a krb5.conf monitor update.
type ConfUpdate struct {
	Config *config.Config
}

// ConfMon is a krb5.conf monitor.
type ConfMon struct {
	confDir  string
	confFile string
	watcher  *fsnotify.Watcher
	config   *config.Config
	updates  chan *ConfUpdate
	done     chan struct{}
}

// sendUpdate sends an update over the updates channel.
func (c *ConfMon) sendUpdate(update *ConfUpdate) {
	// send an update or abort if we are shutting down
	select {
	case c.updates <- update:
	case <-c.done:
	}
}

// isConfigFileEvent checks if event is a config file event.
func (c *ConfMon) isConfigFileEvent(event fsnotify.Event) bool {
	return event.Name == c.confFile
}

// handleConfigFileEvent handles a config file event.
func (c *ConfMon) handleConfigFileEvent(event fsnotify.Event) {
	// check event
	if !c.isConfigFileEvent(event) {
		return
	}
	log.WithFields(log.Fields{
		"name": event.Name,
		"op":   event.Op,
	}).Debug("Kerberos Config Monitor handling file event")

	// load config file
	cfg, err := config.Load(c.confFile)
	if err != nil {
		log.WithError(err).
			Error("Kerberos Config Monitor could not load config")
		return
	}

	// check if config changed
	if reflect.DeepEqual(cfg, c.config) {
		return
	}

	// config changed, send update
	c.config = cfg
	c.sendUpdate(&ConfUpdate{Config: c.config})
}

// handleConfigFileError handles a config file error.
func (c *ConfMon) handleConfigFileError(err error) {
	log.WithError(err).Error("Kerberos Config Monitor watcher error event")
}

// start starts the config monitor.
func (c *ConfMon) start() {
	defer close(c.updates)
	defer func() {
		if err := watcherClose(c.watcher); err != nil {
			log.WithError(err).Error("Kerberos Config Monitor file watcher close error")
		}
	}()

	// handle initial config file
	c.handleConfigFileEvent(fsnotify.Event{Name: c.confFile})

	// watch config file
	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				log.Error("Kerberos Config Monitor got unexpected close of events channel")
				return
			}
			c.handleConfigFileEvent(event)

		case err, ok := <-c.watcher.Errors:
			if !ok {
				log.Error("Kerberos Config Monitor got unexpected close of errors channel")
				return
			}
			c.handleConfigFileError(err)

		case <-c.done:
			return
		}
	}
}

// Start starts the config monitor.
func (c *ConfMon) Start() error {
	// create watcher
	watcher, err := fsnotifyNewWatcher()
	if err != nil {
		log.WithError(err).Error("Kerberos Config Monitor file watcher error")
		return err
	}

	// get config folder and add it to watcher
	if err := watcherAdd(watcher, c.confDir); err != nil {
		log.WithField("dir", c.confDir).WithError(err).Error("Kerberos Config Monitor add CCache error")
		return err
	}

	c.watcher = watcher
	go c.start()

	return nil
}

// Stop stops the config monitor.
func (c *ConfMon) Stop() {
	close(c.done)
	for range c.updates {
		// wait for channel shutdown
	}
}

// Updates returns the channel for config updates.
func (c *ConfMon) Updates() chan *ConfUpdate {
	return c.updates
}

// NewConfMon returns a new krb5.conf monitor.
func NewConfMon() *ConfMon {
	return &ConfMon{
		confFile: krb5conf,
		confDir:  filepath.Dir(krb5conf),
		updates:  make(chan *ConfUpdate),
		done:     make(chan struct{}),
	}
}
