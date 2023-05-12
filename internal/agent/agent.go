package agent

import (
	"time"

	"github.com/T-Systems-MMS/fw-id-agent/internal/api"
	"github.com/T-Systems-MMS/fw-id-agent/internal/client"
	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
	"github.com/T-Systems-MMS/fw-id-agent/internal/krbmon"
	"github.com/T-Systems-MMS/fw-id-agent/internal/notify"
	"github.com/T-Systems-MMS/fw-id-agent/internal/status"
	"github.com/T-Systems-MMS/tnd/pkg/trustnet"
	log "github.com/sirupsen/logrus"
)

// Agent is the firewall identity Agent
type Agent struct {
	config *config.Config
	server *api.Server
	ccache *krbmon.CCacheMon
	krbcfg *krbmon.ConfMon
	tnd    *trustnet.TND
	client *client.Client
	login  chan bool
	done   chan struct{}
	closed chan struct{}

	// last ccache and config update
	ccacheUp *krbmon.CCacheUpdate
	krbcfgUp *krbmon.ConfUpdate

	// kerberos tgt times
	tgtStartTime time.Time
	tgtEndTime   time.Time

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
	if !a.config.Notifications {
		// desktop notifications disabled
		return
	}
	if !a.trusted {
		notify.Notify("No Trusted Network", "No trusted network detected")
		return
	}
	notify.Notify("Trusted Network", "Trusted network detected")
}

// notifyLogin notifies the user if the identity agent is logged in
func (a *Agent) notifyLogin() {
	if !a.config.Notifications {
		// desktop notifications disabled
		return
	}
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
	// make sure client is not already running
	if a.client != nil {
		return
	}

	// make sure ccache is available
	if a.ccacheUp == nil || a.ccacheUp.CCache == nil {
		return
	}

	// make sure kerberos config is available
	if a.krbcfgUp == nil || a.krbcfgUp.Config == nil {
		return
	}

	// start new client
	a.loggedIn = false
	a.client = client.NewClient(a.config, a.ccacheUp.CCache, a.krbcfgUp.Config)
	a.client.Start()
	a.login = a.client.Results()
}

// stopClient stops the client
func (a *Agent) stopClient() {
	// make sure client is running
	if a.client == nil {
		return
	}

	// stop existing client
	a.loggedIn = false
	a.client.Stop()
	a.client = nil
	a.login = nil
}

// handleRequest handles an api request
func (a *Agent) handleRequest(request *api.Request) {
	switch request.Type() {
	case api.TypeQuery:
		// create status
		s := status.New()
		s.TrustedNetwork = a.trusted
		s.LoggedIn = a.loggedIn
		s.Config = a.config
		s.KerberosTGT = status.KerberosTicket{
			StartTime: a.tgtStartTime.Unix(),
			EndTime:   a.tgtEndTime.Unix(),
		}

		// convert status to json and set it as reply
		b, err := s.JSON()
		if err != nil {
			log.WithError(err).Fatal("Agent could not convert status to json")
		}
		request.Reply(b)

		// send reply and close request
		go request.Close()

	case api.TypeRelogin:
		log.Info("Agent got relogin request from user")
		if !a.trusted {
			// no trusted network, abort
			log.Error("Agent not connected to a trusted network, not restarting client")
			request.Error("Not connected to a trusted network")

			// send reply and close request
			go request.Close()

			return
		}

		// trusted network, restart client
		log.Info("Agent is restarting client")
		a.stopClient()
		a.startClient()

		// send reply and close request
		go request.Close()
	}
}

// start starts the agent's main loop
func (a *Agent) start() {
	defer close(a.closed)

	// start api server
	a.server.Start()
	defer a.server.Stop()

	// start ccache monitor
	a.ccache.Start()
	defer a.ccache.Stop()

	// start kerberos config monitor
	a.krbcfg.Start()
	defer a.krbcfg.Stop()

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

		case u, ok := <-a.ccache.Updates():
			if !ok {
				log.Debug("Agent ccache updates channel closed")
				return
			}

			// handle update
			if tgt := u.GetTGT(a.config.Realm); tgt != nil {
				// check if tgt changed
				if tgt.StartTime.Equal(a.tgtStartTime) &&
					tgt.EndTime.Equal(a.tgtEndTime) {
					// tgt did not change
					break
				}

				// tgt changed
				log.WithFields(log.Fields{
					"StartTime": tgt.StartTime,
					"EndTime":   tgt.EndTime,
				}).Debug("Agent got updated kerberos TGT")

				// save start and end time
				a.tgtStartTime = tgt.StartTime
				a.tgtEndTime = tgt.EndTime

				// save update
				a.ccacheUp = u

				// set ccache in existing client or check if we
				// can start new client now
				if a.client != nil {
					a.client.SetCCache(u.CCache)
				}
				if a.trusted {
					a.startClient()
				}
			}

		case u, ok := <-a.krbcfg.Updates():
			if !ok {
				log.Debug("Agent kerberos config updates channel closed")
				return
			}

			// config changed
			log.Debug("Agent got updated kerberos config")

			// save update
			a.krbcfgUp = u

			// set config in existing client or check if we can
			// start a new client now
			if a.client != nil {
				a.client.SetKrb5Conf(u.Config)
			}
			if a.trusted {
				a.startClient()
			}

		case r, ok := <-a.server.Requests():
			if !ok {
				log.Debug("Agent server requests channel closed")
				return
			}
			a.handleRequest(r)

		case sleep, ok := <-sleepMon.Events():
			if !ok {
				log.Debug("Agent SleepMon events channel closed")
				return
			}

			// ignore wake-up event
			if !sleep {
				break
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
	server := api.NewServer(api.GetUserSocketFile())
	ccache := krbmon.NewCCacheMon()
	krbcfg := krbmon.NewConfMon()
	tnd := trustnet.NewTND()
	return &Agent{
		config: config,
		server: server,
		ccache: ccache,
		krbcfg: krbcfg,
		tnd:    tnd,
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}
