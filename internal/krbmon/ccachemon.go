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
	cCache     *credentials.CCache
	updates    chan *CCacheUpdate
	done       chan struct{}
}

// sendUpdate sends an update over the updates channel.
func (c *CCacheMon) sendUpdate(update *CCacheUpdate) {
	// send an update or abort if we are shutting down
	select {
	case c.updates <- update:
	case <-c.done:
	}
}

// createCredentialCacheEnvVar creates an expected environment variable value
// for the credential cache based on the current user ID.
func createCredentialCacheEnvVar() string {
	osUser, err := user.Current()
	if err != nil {
		log.WithError(err).
			Error("Kerberos CCache Monitor could not create credential cache environment variable value")
		return ""
	}
	return fmt.Sprintf("FILE:/tmp/krb5cc_%s", osUser.Uid)
}

// getCredentialsCacheFilename returns the ccache file name.
func getCredentialCacheFilename() (string, error) {
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
func (c *CCacheMon) handleCCacheFileEvent() {
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

// start starts the ccache monitor.
func (c *CCacheMon) start() {
	defer close(c.updates)

	// get ccache file
	cCacheFile, err := getCredentialCacheFilename()
	if err != nil {
		log.WithError(err).Fatal("Kerberos CCache Monitor could not get CCache file")
	}
	c.cCacheFile = cCacheFile

	// get directory of ccache file
	c.cCacheDir = filepath.Dir(cCacheFile)
	if c.cCacheDir == "" {
		log.Fatal("Kerberos CCache Monitor could not get CCache dir")
	}

	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Fatal("Kerberos CCache Monitor file watcher error")
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.WithError(err).Error("Kerberos CCache Monitor file watcher close error")
		}
	}()

	// add ccache folder to watcher
	if err := watcher.Add(c.cCacheDir); err != nil {
		log.WithError(err).Debug("Kerberos CCache Monitor add CCache error")
		return
	}

	// handle initial ccache file
	c.handleCCacheFileEvent()

	// watch ccache file
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Error("Kerberos CCache Monitor got unexpected close of events channel")
				return
			}
			if c.isCCacheFileEvent(event) {
				log.WithFields(log.Fields{
					"name": event.Name,
					"op":   event.Op,
				}).Debug("Kerberos CCache Monitor handling file event")
				c.handleCCacheFileEvent()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				log.Error("Kerberos CCache Monitor got unexpected close of errors channel")
				return
			}
			log.WithError(err).Error("Kerberos CCache Monitor watcher error event")

		case <-c.done:
			return
		}
	}
}

// Start starts the ccache monitor.
func (c *CCacheMon) Start() {
	go c.start()
}

// Stop stops the ccache monitor.
func (c *CCacheMon) Stop() {
	close(c.done)
	for range c.updates {
		// wait for channel shutdown
	}
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
	}
}
