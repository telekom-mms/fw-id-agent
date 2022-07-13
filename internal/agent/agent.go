package agent

import (
	"github.com/T-Systems-MMS/fw-id-agent/internal/client"
	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
	"github.com/T-Systems-MMS/fw-id-agent/internal/notify"
	"github.com/T-Systems-MMS/tnd/pkg/trustnet"
	log "github.com/sirupsen/logrus"
)

// Agent is the firewall identity Agent
type Agent struct {
	config *config.Config
	tnd    *trustnet.TND
	client *client.Client
	login  chan bool
	done   chan struct{}
	closed chan struct{}
}

// notifyTND notifies the user if we are connected to a trusted network
func (a *Agent) notifyTND(trusted bool) {
	if !trusted {
		notify.Notify("No Trusted Network", "No trusted network detected")
		return
	}
	notify.Notify("Trusted Network", "Trusted network detected")
}

// notifyLogin notifies the user if the identity agent is logged in
func (a *Agent) notifyLogin(loggedIn bool) {
	if !loggedIn {
		notify.Notify("Identity Agent Logout", "Identity Agent logged out")
		return
	}
	notify.Notify("Identity Agent Login", "Identity Agent logged in successfully")
}

// initTND initializes the trusted network detection from the config
func (a *Agent) initTND() {
	// add https servers
	for _, s := range a.config.TND.HTTPSServers {
		log.WithFields(log.Fields{
			"url":  s.URL,
			"hash": s.Hash,
		}).Debug("Agent adding HTTPS server url and hash to TND")
		a.tnd.AddServer(s.URL, s.Hash)
	}
}

// initClient initializes the identity agent client from the config
func (a *Agent) initClient() {
	a.client.SetURL(a.config.ServiceURL)
	a.client.SetRealm(a.config.Realm)
}

// startClient starts the client
func (a *Agent) startClient() {
	if a.client != nil {
		return
	}
	a.client = client.NewClient()
	a.initClient()
	a.client.Start()
	a.login = a.client.Results()
}

// stopClient stops the client
func (a *Agent) stopClient() {
	if a.client == nil {
		return
	}
	a.client.Stop()
	a.client = nil
	a.login = nil
}

// start starts the agent's main loop
func (a *Agent) start() {
	defer close(a.closed)

	// start trusted network detection
	a.initTND()
	a.tnd.Start()
	defer a.tnd.Stop()

	// start main loop
	trusted := false
	loggedIn := false
	for {
		select {
		case r, ok := <-a.tnd.Results():
			if !ok {
				log.Debug("Agent TND results channel closed")
				return
			}

			// check if trusted state changed
			if r != trusted {
				log.WithField("trusted", r).
					Debug("Agent got trusted network change")
				trusted = r
				a.notifyTND(trusted)
				if trusted {
					// switched to trusted network,
					// start identity agent client
					a.startClient()
				} else {
					// switched to untrusted network,
					// stop identity agent client
					a.stopClient()
				}
			}

		case r, ok := <-a.login:
			if !ok {
				log.Debug("Agent client results channel closed")
				return
			}

			// check if logged in state changed
			if r != loggedIn {
				log.WithField("loggedIn", r).Debug("Agent got login change")
				loggedIn = r
				a.notifyLogin(loggedIn)
			}

		case <-a.done:
			log.Debug("Agent stopping")
			a.stopClient()
			return
		}
	}
}

// Start starts the agent
func (a *Agent) Start() {
	go a.start()
}

// Stop stops the agent
func (a *Agent) Stop() {
	close(a.done)
	<-a.closed
}

// NewAgent returns a new agent
func NewAgent(config *config.Config) *Agent {
	tnd := trustnet.NewTND()
	return &Agent{
		config: config,
		tnd:    tnd,
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}
