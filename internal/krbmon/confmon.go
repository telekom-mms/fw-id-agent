package krbmon

import (
	"path/filepath"
	"reflect"

	"github.com/fsnotify/fsnotify"
	"github.com/jcmturner/gokrb5/v8/config"
	log "github.com/sirupsen/logrus"
)

const (
	// krb5conf is the file path of the krb5.conf file
	krb5conf = "/etc/krb5.conf"
)

// ConfUpdate is a krb5.conf monitor update
type ConfUpdate struct {
	Config *config.Config
}

// ConfMon is a krb5.conf monitor
type ConfMon struct {
	confDir  string
	confFile string
	config   *config.Config
	updates  chan *ConfUpdate
	done     chan struct{}
}

// sendUpdate sends an update over the updates channel
func (c *ConfMon) sendUpdate(update *ConfUpdate) {
	// send an update or abort if we are shutting down
	select {
	case c.updates <- update:
	case <-c.done:
	}
}

// isConfigFileEvent checks if event is a config file event
func (c *ConfMon) isConfigFileEvent(event fsnotify.Event) bool {
	if event.Name == c.confFile {
		return true
	}
	return false
}

// handleConfigFileEvent handles a config file event
func (c *ConfMon) handleConfigFileEvent() {
	// load config file
	cfg, err := config.Load(krb5conf)
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

// start starts the config monitor
func (c *ConfMon) start() {
	defer close(c.updates)

	// get directory of ccache file
	c.confDir = filepath.Dir(krb5conf)
	if c.confDir == "" {
		log.Fatal("Kerberos Config Monitor could not get config dir")
	}

	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Fatal("Kerberos Config Monitor file watcher error")
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.WithError(err).Error("Kerberos Config Monitor file watcher close error")
		}
	}()

	// add config folder to watcher
	if err := watcher.Add(c.confDir); err != nil {
		log.WithError(err).Debug("Kerberos Config Monitor add CCache error")
		return
	}

	// handle initial config file
	c.handleConfigFileEvent()

	// watch config file
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Error("Kerberos Config Monitor got unexpected close of events channel")
				return
			}
			if c.isConfigFileEvent(event) {
				log.WithFields(log.Fields{
					"name": event.Name,
					"op":   event.Op,
				}).Debug("Kerberos Config Monitor handling file event")
				c.handleConfigFileEvent()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				log.Error("Kerberos Config Monitor got unexpected close of errors channel")
				return
			}
			log.WithError(err).Error("Kerberos Config Monitor watcher error event")

		case <-c.done:
			return
		}
	}
}

// Start starts the config monitor
func (c *ConfMon) Start() {
	go c.start()
}

// Stop stops the config monitor
func (c *ConfMon) Stop() {
	close(c.done)
	for range c.updates {
		// wait for channel shutdown
	}
}

// Updates returns the channel for config updates
func (c *ConfMon) Updates() chan *ConfUpdate {
	return c.updates
}

// NewConfMon returns a new krb5.conf monitor
func NewConfMon() *ConfMon {
	return &ConfMon{
		updates: make(chan *ConfUpdate),
		done:    make(chan struct{}),
	}
}
