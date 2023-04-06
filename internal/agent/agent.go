package agent

import (
	"github.com/T-Systems-MMS/fw-id-agent/internal/api"
	"github.com/T-Systems-MMS/fw-id-agent/internal/client"
	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
	"github.com/T-Systems-MMS/fw-id-agent/internal/notify"
	"github.com/T-Systems-MMS/fw-id-agent/internal/status"
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

	// trusted network and login status
	trusted  bool
	loggedIn bool
}

// logTND logs if we are connected to a trusted network
func (a *Agent) logTND() {
	if !a.trusted {
		log.Info("Agent is not connected to a trusted network")
		return
	}
	log.Info("Agent is connected to a trusted network")
}

// logLogin logs if the identity agent is logged in
func (a *Agent) logLogin() {
	if !a.loggedIn {
		log.Info("Agent logged out")
		return
	}
	log.Info("Agent logged in successfully")
}

// notifyTND notifies the user if we are connected to a trusted network
func (a *Agent) notifyTND() {
	if !a.trusted {
		notify.Notify("No Trusted Network", "No trusted network detected")
		return
	}
	notify.Notify("Trusted Network", "Trusted network detected")
}

// notifyLogin notifies the user if the identity agent is logged in
func (a *Agent) notifyLogin() {
	if !a.loggedIn {
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

// startClient starts the client
func (a *Agent) startClient() {
	if a.client != nil {
		return
	}
	a.loggedIn = false
	a.client = client.NewClient(a.config)
	a.client.Start()
	a.login = a.client.Results()
}

// stopClient stops the client
func (a *Agent) stopClient() {
	if a.client == nil {
		return
	}
	a.loggedIn = false
	a.client.Stop()
	a.client = nil
	a.login = nil
}

// handleRequest handles an api request
func (a *Agent) handleRequest(request *api.Request) {
	switch request.Type() {
	case api.TypeQuery:
		status := status.New()
		status.TrustedNetwork = a.trusted
		status.LoggedIn = a.loggedIn
		status.Config = a.config
		b, err := status.JSON()
		if err != nil {
			log.WithError(err).Fatal("Agent could not convert status to json")
		}
		request.Reply(b)
		go request.Close()
	case api.TypeRelogin:
		log.Info("Agent got relogin request from user")
		if !a.trusted {
			// no trusted network, abort
			log.Error("Agent not connected to a trusted network, not restarting client")
			request.Error("Not connected to a trusted network")
			go request.Close()
			return
		}

		// trusted network, restart client
		log.Info("Agent is restarting client")
		a.stopClient()
		a.startClient()
		go request.Close()
	}
}

// start starts the agent's main loop
func (a *Agent) start() {
	defer close(a.closed)

	// start api server
	server := api.NewServer(api.GetUserSocketFile())
	server.Start()
	defer server.Stop()

	// start trusted network detection
	a.initTND()
	a.tnd.Start()
	defer a.tnd.Stop()

	// start sleep monitor
	sleepMon := NewSleepMon()
	sleepMon.Start()
	defer sleepMon.Stop()

	// start main loop
	for {
		select {
		case r, ok := <-a.tnd.Results():
			if !ok {
				log.Debug("Agent TND results channel closed")
				return
			}

			// check if trusted state changed
			if r != a.trusted {
				log.WithField("trusted", r).
					Debug("Agent got trusted network change")
				a.trusted = r
				a.logTND()
				a.notifyTND()
				if a.trusted {
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
			if r != a.loggedIn {
				log.WithField("loggedIn", r).Debug("Agent got login change")
				a.loggedIn = r
				a.logLogin()
				a.notifyLogin()
			}

		case r, ok := <-server.Requests():
			if !ok {
				log.Debug("Agent server requests channel closed")
				return
			}
			a.handleRequest(r)

		case _, ok := <-sleepMon.Events():
			if !ok {
				log.Debug("Agent SleepMon events channel closed")
				return
			}

			// reset trusted network status and stop client
			log.Info("Agent got sleep event, resetting trusted network status and stopping client")
			a.trusted = false
			a.stopClient()

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
