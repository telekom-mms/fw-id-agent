// Package krbmon contains kerberos monitoring components.
package krbmon

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/iana/nametype"
	"github.com/jcmturner/gokrb5/v8/types"
	log "github.com/sirupsen/logrus"
)

// CCacheUpdate is a ccache monitor update.
type CCacheUpdate struct {
	CCache *credentials.CCache
}

// GetTGT returns the TGT for realm in the ccache.
func (u *CCacheUpdate) GetTGT(realm string) *credentials.Credential {
	name := types.NewPrincipalName(nametype.KRB_NT_SRV_INST, "krbtgt/"+realm)
	if tgt, ok := u.CCache.GetEntry(name); ok {
		return tgt
	}
	return nil
}

// CCacheMon is a ccache monitor.
type CCacheMon struct {
	cCacheDir  string
	cCacheFile string
	watcher    *fsnotify.Watcher
	cCache     *credentials.CCache
	updates    chan *CCacheUpdate
	done       chan struct{}
	closed     chan struct{}
}

// sendUpdate sends an update over the updates channel.
func (c *CCacheMon) sendUpdate(update *CCacheUpdate) {
	// send an update or abort if we are shutting down
	select {
	case c.updates <- update:
	case <-c.done:
	}
}

// userCurrent is user.Current for testing.
var userCurrent = user.Current

// createCredentialCacheEnvVar creates an expected environment variable value
// for the credential cache based on the current user ID.
func createCredentialCacheEnvVar() string {
	osUser, err := userCurrent()
	if err != nil {
		log.WithError(err).
			Error("Kerberos CCache Monitor could not create credential cache environment variable value")
		return ""
	}
	return fmt.Sprintf("FILE:/tmp/krb5cc_%s", osUser.Uid)
}

// getCredentialCacheFilename returns the ccache file name.
var getCredentialCacheFilename = func() (string, error) {
	envVar := os.Getenv("KRB5CCNAME")
	if envVar == "" {
		newEnv := createCredentialCacheEnvVar()
		log.WithField("new", newEnv).
			Debug("Kerberos CCache Monitor could not get environment variable KRB5CCNAME, setting it")
		envVar = newEnv
	}
	if !strings.HasPrefix(envVar, "FILE:") {
		newEnv := createCredentialCacheEnvVar()
		log.WithFields(log.Fields{
			"old": envVar,
			"new": newEnv,
		}).Error("Kerberos CCache Monitor got invalid environment variable KRB5CCNAME, resetting it")
		envVar = newEnv
	}
	if envVar == "" {
		// environment variable still invalid
		return "", fmt.Errorf("environment variable KRB5CCNAME is not set")
	}
	return strings.TrimPrefix(envVar, "FILE:"), nil
}

// isCCacheFileEvent checks if event is a ccache file event.
func (c *CCacheMon) isCCacheFileEvent(event fsnotify.Event) bool {
	return event.Name == c.cCacheFile
}

// handleCCacheFileEvent handles a ccache file event.
func (c *CCacheMon) handleCCacheFileEvent(event fsnotify.Event) {
	// check event
	if !c.isCCacheFileEvent(event) {
		return
	}
	log.WithFields(log.Fields{
		"name": event.Name,
		"op":   event.Op,
	}).Debug("Kerberos CCache Monitor handling file event")

	// read ccache file
	b, err := os.ReadFile(c.cCacheFile)
	if err != nil {
		log.WithError(err).Error("Kerberos CCache Monitor could not read credential cache file")
		return
	}

	// check file length to make sure loading the credential cache below
	// does not fail. this is a rough estimate of a minimum ccache file
	// that contains: the version indicator (2 bytes), no header, minimum
	// default principal (8 bytes), one minimum credential (59 bytes). See
	// https://web.mit.edu/kerberos/krb5-devel/doc/formats/ccache_file_format.html
	if len(b) < 69 {
		log.Error("Kerberos CCache Monitor read invalid credential cache file")
		return
	}

	// load ccache
	cCache := new(credentials.CCache)
	err = cCache.Unmarshal(b)
	if err != nil {
		log.WithError(err).Error("Kerberos CCache Monitor could not load credential cache")
		return
	}

	// check if ccache changed
	if reflect.DeepEqual(cCache, c.cCache) {
		return
	}

	// ccache changed, send update
	c.cCache = cCache
	c.sendUpdate(&CCacheUpdate{CCache: c.cCache})
}

// handleCCacheFileError handles a ccache file error.
func (c *CCacheMon) handleCCacheFileError(err error) {
	log.WithError(err).Error("Kerberos CCache Monitor watcher error event")
}

// start starts the ccache monitor.
func (c *CCacheMon) start() {
	defer close(c.closed)
	defer close(c.updates)
	defer func() {
		if err := watcherClose(c.watcher); err != nil {
			log.WithError(err).Error("Kerberos CCache Monitor file watcher close error")
		}
	}()

	// handle initial ccache file
	c.handleCCacheFileEvent(fsnotify.Event{Name: c.cCacheFile})

	// watch ccache file
	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				log.Error("Kerberos CCache Monitor got unexpected close of events channel")
				return
			}
			c.handleCCacheFileEvent(event)

		case err, ok := <-c.watcher.Errors:
			if !ok {
				log.Error("Kerberos CCache Monitor got unexpected close of errors channel")
				return
			}
			c.handleCCacheFileError(err)

		case <-c.done:
			return
		}
	}
}

// Start starts the ccache monitor.
func (c *CCacheMon) Start() error {
	// get ccache file
	cCacheFile, err := getCredentialCacheFilename()
	if err != nil {
		log.WithError(err).Error("Kerberos CCache Monitor could not get CCache file")
		return err
	}
	c.cCacheFile = cCacheFile

	// create watcher
	watcher, err := fsnotifyNewWatcher()
	if err != nil {
		log.WithError(err).Error("Kerberos CCache Monitor file watcher error")
		return err
	}

	// get ccache folder and add it to watcher
	c.cCacheDir = filepath.Dir(cCacheFile)
	if err := watcherAdd(watcher, c.cCacheDir); err != nil {
		log.WithField("dir", c.cCacheDir).WithError(err).Error("Kerberos CCache Monitor add CCache error")
		return err
	}

	c.watcher = watcher
	go c.start()

	return nil
}

// Stop stops the ccache monitor.
func (c *CCacheMon) Stop() {
	close(c.done)
	<-c.closed
}

// Updates returns the channel for ccache updates.
func (c *CCacheMon) Updates() chan *CCacheUpdate {
	return c.updates
}

// NewCCacheMon returns a new ccache monitor.
func NewCCacheMon() *CCacheMon {
	return &CCacheMon{
		updates: make(chan *CCacheUpdate),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
}
