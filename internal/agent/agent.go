package agent

import (
	"errors"
	"time"

	"github.com/T-Systems-MMS/fw-id-agent/internal/api"
	"github.com/T-Systems-MMS/fw-id-agent/internal/client"
	"github.com/T-Systems-MMS/fw-id-agent/internal/dbusapi"
	"github.com/T-Systems-MMS/fw-id-agent/internal/krbmon"
	"github.com/T-Systems-MMS/fw-id-agent/internal/notify"
	"github.com/T-Systems-MMS/fw-id-agent/pkg/config"
	"github.com/T-Systems-MMS/fw-id-agent/pkg/status"
	"github.com/T-Systems-MMS/tnd/pkg/trustnet"
	log "github.com/sirupsen/logrus"
)

// Agent is the firewall identity Agent
type Agent struct {
	config *config.Config
	server *api.Server
	dbus   *dbusapi.Service
	ccache *krbmon.CCacheMon
	krbcfg *krbmon.ConfMon
	tnd    *trustnet.TND
	client *client.Client
	login  chan status.LoginState
	done   chan struct{}
	closed chan struct{}

	// last ccache and config update
	ccacheUp *krbmon.CCacheUpdate
	krbcfgUp *krbmon.ConfUpdate

	// kerberos tgt times
	kerberosTGT status.KerberosTicket

	// trusted network and login status
	trustedNetwork status.TrustedNetwork
	loginState     status.LoginState
	loggedIn       bool

	// last client keep-alive
	lastKeepAlive int64
}

// logTND logs if we are connected to a trusted network
func (a *Agent) logTND() {
	if !a.trustedNetwork.Trusted() {
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
	if !a.trustedNetwork.Trusted() {
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

// handleKerberosTGTChange handles a change of the kerberos TGT times
func (a *Agent) handleKerberosTGTChange() {
	log.WithFields(log.Fields{
		"StartTime": a.kerberosTGT.StartTime,
		"EndTime":   a.kerberosTGT.EndTime,
	}).Info("Kerberos TGT times changed")
	a.dbus.SetProperty(dbusapi.PropertyKerberosTGTStartTime, a.kerberosTGT.StartTime)
	a.dbus.SetProperty(dbusapi.PropertyKerberosTGTEndTime, a.kerberosTGT.EndTime)
}

// handleTrustedNetworkChange handles a change of the trusted network status
func (a *Agent) handleTrustedNetworkChange() {
	log.WithField("trustedNetwork", a.trustedNetwork).
		Info("Trusted network status changed")
	a.logTND()
	a.notifyTND()
	a.dbus.SetProperty(dbusapi.PropertyTrustedNetwork, a.trustedNetwork)
}

// handleLoginStateChange handles a change of the login state
func (a *Agent) handleLoginStateChange() {
	log.WithField("loginState", a.loginState).
		Info("Login state changed")

	// if we switched from "logged in" to "logged out" or from "logged out"
	// to "logged in" log the change and notify user
	switch a.loginState {
	case status.LoginStateLoggedOut:
		if a.loggedIn {
			a.loggedIn = false
			a.logLogin()
			a.notifyLogin()
		}
	case status.LoginStateLoggedIn:
		if !a.loggedIn {
			a.loggedIn = true
			a.logLogin()
			a.notifyLogin()
		}
	}

	// set d-bus property
	a.dbus.SetProperty(dbusapi.PropertyLoginState, a.loginState)
}

// handleLastKeepAliveChange handles a change of the last keep-alive time
func (a *Agent) handleLastKeepAliveChange() {
	log.WithField("lastKeepAlive", a.lastKeepAlive).
		Info("Last keep-alive time changed")
	a.dbus.SetProperty(dbusapi.PropertyLastKeepAliveAt, a.lastKeepAlive)
}

// setKerberosTGT sets the kerberos TGT times
func (a *Agent) setKerberosTGT(startTime, endTime int64) {
	if startTime == a.kerberosTGT.StartTime &&
		endTime == a.kerberosTGT.EndTime {
		// ticket not changed
		return
	}

	// ticket changed
	a.kerberosTGT.StartTime = startTime
	a.kerberosTGT.EndTime = endTime
	a.handleKerberosTGTChange()
}

// setTrustedNetwork sets the trusted network status to "trusted" or "not trusted"
func (a *Agent) setTrustedNetwork(trusted bool) {
	// convert bool to trusted network status
	trustedNetwork := status.TrustedNetworkNotTrusted
	if trusted {
		trustedNetwork = status.TrustedNetworkTrusted
	}

	// check status change
	if trustedNetwork == a.trustedNetwork {
		// status not changed
		return
	}

	// status changed
	a.trustedNetwork = trustedNetwork
	a.handleTrustedNetworkChange()
}

// setLoginState sets the login state
func (a *Agent) setLoginState(loginState status.LoginState) {
	if loginState == a.loginState {
		// state not changed
		return
	}

	// state changed
	a.loginState = loginState
	a.handleLoginStateChange()
}

// setLastKeepAlive sets LastKeepAlive
func (a *Agent) setLastKeepAlive(lastKeepAlive int64) {
	if lastKeepAlive == a.lastKeepAlive {
		// timestamp not changed
		return
	}

	// timestamp changed
	a.lastKeepAlive = lastKeepAlive
	a.handleLastKeepAliveChange()
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
	a.setLoginState(status.LoginStateLoggingOut)
	a.client.Stop()
	a.client = nil
	a.login = nil
	a.setLoginState(status.LoginStateLoggedOut)
}

// handleRequest handles an api request
func (a *Agent) handleRequest(request *api.Request) {
	switch request.Type() {
	case api.TypeQuery:
		// create status
		s := status.New()
		s.Config = a.config
		s.TrustedNetwork = a.trustedNetwork
		s.LoginState = a.loginState
		s.LastKeepAlive = a.lastKeepAlive
		s.KerberosTGT = a.kerberosTGT

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
		if !a.trustedNetwork.Trusted() {
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

// handleDBusRequest handles a D-Bus API request
func (a *Agent) handleDBusRequest(request *dbusapi.Request) {
	defer request.Close()

	switch request.Name {
	case dbusapi.RequestReLogin:
		log.Info("Agent got relogin request from user via D-Bus")
		if !a.trustedNetwork.Trusted() {
			// no trusted network, abort
			log.Error("Agent not connected to a trusted network, not restarting client")
			request.Error = errors.New("not connected to a trusted network")
			return
		}

		// trusted network, restart client
		log.Info("Agent is restarting client")
		a.stopClient()
		a.startClient()
	}
}

// start starts the agent's main loop
func (a *Agent) start() {
	defer close(a.closed)

	// start api server
	a.server.Start()
	defer a.server.Stop()

	// start dbus api
	a.dbus.Start()
	defer a.dbus.Stop()

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

	// set trusted network status to "not trusted" and
	// login state to "logged out"
	a.setTrustedNetwork(false)
	a.setLoginState(status.LoginStateLoggedOut)

	// set config D-Bus property
	b, err := a.config.JSON()
	if err != nil {
		log.WithError(err).Fatal("could not convert config to json")
	}
	a.dbus.SetProperty(dbusapi.PropertyConfig, string(b))

	// start main loop
	for {
		select {
		case r, ok := <-a.tnd.Results():
			if !ok {
				log.Debug("Agent TND results channel closed")
				return
			}

			// update trusted network status
			a.setTrustedNetwork(r)

			if a.trustedNetwork.Trusted() {
				// switched to trusted network,
				// start identity agent client
				a.startClient()
			} else {
				// switched to untrusted network,
				// stop identity agent client
				a.stopClient()
			}

		case r, ok := <-a.login:
			if !ok {
				log.Debug("Agent client results channel closed")
				return
			}

			// update login state
			a.setLoginState(r)

			// update last keep-alive
			if r.LoggedIn() {
				now := time.Now().Unix()
				a.setLastKeepAlive(now)
			}

		case u, ok := <-a.ccache.Updates():
			if !ok {
				log.Debug("Agent ccache updates channel closed")
				return
			}

			// handle update
			tgt := u.GetTGT(a.config.Realm)
			if tgt == nil {
				break
			}

			// get start and end unix timestamps of tgt
			startTime := tgt.StartTime.Unix()
			endTime := tgt.EndTime.Unix()

			// check if tgt changed
			if a.kerberosTGT.TimesEqual(startTime, endTime) {
				// tgt did not change
				break
			}

			// tgt changed
			// save start and end time
			a.setKerberosTGT(startTime, endTime)

			// save update
			a.ccacheUp = u

			// set ccache in existing client or check if we
			// can start new client now
			if a.client != nil {
				a.client.SetCCache(u.CCache)
			}
			if a.trustedNetwork.Trusted() {
				a.startClient()
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
			if a.trustedNetwork.Trusted() {
				a.startClient()
			}

		case r, ok := <-a.server.Requests():
			if !ok {
				log.Debug("Agent server requests channel closed")
				return
			}
			a.handleRequest(r)

		case r, ok := <-a.dbus.Requests():
			if !ok {
				log.Debug("Agent dbus requests channel closed")
				return
			}
			a.handleDBusRequest(r)

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
			a.setTrustedNetwork(false)
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
	dbus := dbusapi.NewService()
	ccache := krbmon.NewCCacheMon()
	krbcfg := krbmon.NewConfMon()
	tnd := trustnet.NewTND()
	return &Agent{
		config: config,
		server: server,
		dbus:   dbus,
		ccache: ccache,
		krbcfg: krbcfg,
		tnd:    tnd,
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}
