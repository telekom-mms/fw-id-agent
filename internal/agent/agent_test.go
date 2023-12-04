package agent

import (
	"encoding/hex"
	"reflect"
	"testing"

	krbconfig "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/test/testdata"
	"github.com/telekom-mms/fw-id-agent/internal/client"
	"github.com/telekom-mms/fw-id-agent/internal/dbusapi"
	"github.com/telekom-mms/fw-id-agent/internal/krbmon"
	"github.com/telekom-mms/fw-id-agent/pkg/config"
	"github.com/telekom-mms/fw-id-agent/pkg/status"
)

// nopDBusService is a NOP D-Bus Service for testing.
type nopDBusService struct{}

func (n *nopDBusService) Start()                          {}
func (n *nopDBusService) Stop()                           {}
func (n *nopDBusService) Requests() chan *dbusapi.Request { return nil }
func (n *nopDBusService) SetProperty(string, any)         {}

// TestAgentSetKerberosTGT tests setKerberosTGT of Agent.
func TestAgentSetKerberosTGT(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// test values
	for i, want := range []status.KerberosTicket{
		// set ticket, set new value
		{StartTime: 1, EndTime: 2},
		// set ticket again, no change
		{StartTime: 1, EndTime: 2},
	} {
		a.setKerberosTGT(want.StartTime, want.EndTime)

		// check values
		got := a.kerberosTGT
		if !reflect.DeepEqual(got, want) {
			t.Errorf("test %d: got %v, want %v", i, got, want)
		}
	}
}

// TestAgentSetTrustedNetwork tests setTrustedNetwork of Agent.
func TestAgentSetTrustedNetwork(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// test values
	for i, noti := range []bool{
		// disable notifications
		false,
		// enable notifications,
		true,
	} {
		a.config.Notifications = noti

		for j, want := range []bool{
			// set trusted, set new value
			true,
			// set trusted again, no change
			true,
			// set untrusted, set new value
			false,
		} {
			a.setTrustedNetwork(want)

			// check values
			got := a.trustedNetwork.Trusted()
			if got != want {
				t.Errorf("test %d, %d: got %t, want %t", i, j,
					got, want)
			}
		}
	}
}

// TestAgentSetLoginState tests setLoginState of Agent.
func TestAgentSetLoginState(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// test values
	for i, noti := range []bool{
		// disable notifications
		false,
		// enable notifications,
		true,
	} {
		a.config.Notifications = noti

		for j, want := range []status.LoginState{
			// set logged in, set new value
			status.LoginStateLoggedIn,
			// set logged in again, no change
			status.LoginStateLoggedIn,
			// set logged out, set new value
			status.LoginStateLoggedOut,
		} {
			a.setLoginState(want)

			// check values
			got := a.loginState
			if got != want {
				t.Errorf("test %d, %d: got %v, want %v", i, j,
					got, want)
			}
		}
	}
}

// TestAgentSetLastKeepAlive tests setLastKeepAlive of Agent.
func TestAgentSetLastKeepAlive(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// test values
	for i, want := range []int64{
		// set last keep-alive, set new value
		1,
		// set last keep-alive again, no change
		1,
	} {
		a.setLastKeepAlive(want)

		// check values
		got := a.lastKeepAlive
		if got != want {
			t.Errorf("test %d: got %v, want %v", i, got, want)
		}
	}
}

// TestInitTND tests initTND of Agent.
func TestInitTND(_ *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)

	// no https servers
	// TODO: add checks when TND has getters
	a.initTND()

	// with https servers
	a.config.TND.HTTPSServers = []config.TNDHTTPSConfig{
		{URL: "example.com", Hash: "abcdef1234567890"},
	}
	a.initTND()
}

// TestAgentStartStopClient tests startClient and stopClient of Agent.
func TestAgentStartStopClient(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// ccache not set
	a.startClient()
	if a.client != nil {
		t.Errorf("client should not run without ccache")
	}

	// set ccache
	b, err := hex.DecodeString(testdata.CCACHE_TEST)
	if err != nil {
		t.Fatal(err)
	}
	ccache := new(credentials.CCache)
	err = ccache.Unmarshal(b)
	if err != nil {
		t.Fatal(err)
	}
	ccacheUp := &krbmon.CCacheUpdate{CCache: ccache}
	a.ccacheUp = ccacheUp

	// conf not set
	a.startClient()
	if a.client != nil {
		t.Errorf("client should not run without conf")
	}

	// set conf
	confUp := &krbmon.ConfUpdate{Config: krbconfig.New()}
	a.krbcfgUp = confUp

	// all set, start client
	a.startClient()
	if a.client == nil {
		t.Errorf("client should run")
	}

	// shut down
	a.stopClient()
	if a.client != nil {
		t.Errorf("client should not run after stop")
	}
}

// TestAgentHandleTNDResult tests handleTNDResult of Agent.
func TestAgentHandleTNDResult(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// test trusted
	a.handleTNDResult(true)
	if !a.trustedNetwork.Trusted() {
		t.Error("network should be trusted")
	}

	// test not trusted
	a.handleTNDResult(false)
	if a.trustedNetwork.Trusted() {
		t.Error("network should not be trusted")
	}
}

// TestAgentHandleLoginResult tests handleLoginResult of Agent.
func TestAgentHandleLoginResult(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// test logged in
	a.handleLoginResult(status.LoginStateLoggedIn)
	if !a.loginState.LoggedIn() {
		t.Error("client should be logged in")
	}

	// test logged out
	a.handleLoginResult(status.LoginStateLoggedOut)
	if a.loginState != status.LoginStateLoggedOut {
		t.Error("client should be logged out")
	}
}

// TestAgentHandleCCacheUpdate tests handleCCacheUpdate of Agent.
func TestAgentHandleCCacheUpdate(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// create ccache update
	b, err := hex.DecodeString(testdata.CCACHE_TEST)
	if err != nil {
		t.Fatal(err)
	}
	ccache := new(credentials.CCache)
	err = ccache.Unmarshal(b)
	if err != nil {
		t.Fatal(err)
	}
	ccacheUp := &krbmon.CCacheUpdate{CCache: ccache}

	// set network to trusted
	a.client = &client.Client{}
	a.trustedNetwork = status.TrustedNetworkTrusted

	// test wrong realm/no tgt
	a.handleCCacheUpdate(ccacheUp)
	if a.ccacheUp != nil {
		t.Error("ccache update should not be set")
	}

	// test right realm/with tgt
	a.config.Realm = "TEST.GOKRB5"
	a.handleCCacheUpdate(ccacheUp)
	if a.ccacheUp != ccacheUp {
		t.Error("ccache update should be set")
	}

	// test double update/update without changes
	a.handleCCacheUpdate(ccacheUp)
	if a.ccacheUp != ccacheUp {
		t.Error("ccache update should still be set")
	}
}

// TestAgentHandleKrbConfUpdate tests handleKrbConfUpdate of Agent.
func TestAgentHandleKrbConfUpdate(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}
	a.client = &client.Client{}
	a.trustedNetwork = status.TrustedNetworkTrusted

	// test update
	u := &krbmon.ConfUpdate{}
	a.handleKrbConfUpdate(u)
	if a.krbcfgUp != u {
		t.Error("krb5.conf update should be set")
	}
}

// TestAgentHandleDBusRequest tests handleDBusRequest of Agent.
func TestAgentHandleDBusRequest(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}

	// test network not trusted
	request := dbusapi.NewRequest(dbusapi.RequestReLogin, nil)
	a.handleDBusRequest(request)
	request.Wait()
	if request.Error == nil {
		t.Error("request should have failed and error should be set")
	}

	// network trusted
	a.trustedNetwork = status.TrustedNetworkTrusted
	request = dbusapi.NewRequest(dbusapi.RequestReLogin, nil)
	a.handleDBusRequest(request)
	request.Wait()
	if request.Error != nil {
		t.Error("request should be OK and and error should not be set")
	}
}

// TrestAgentHandleSleepEvent tests handleSleepEvent of Agent.
func TestAgentHandleSleepEvent(t *testing.T) {
	// create agent
	c := &config.Config{}
	a := NewAgent(c)
	a.dbus = &nopDBusService{}
	a.client = client.NewClient(a.config, nil, nil)
	a.trustedNetwork = status.TrustedNetworkTrusted
	a.client.Start()

	// test wake-up event, should be ignored
	a.handleSleepEvent(false)
	if a.client == nil || !a.trustedNetwork.Trusted() {
		t.Error("client and trusted network should not be changed")
	}

	// test sleep event, should stop client and reset trusted network
	a.handleSleepEvent(true)
	if a.client != nil && a.trustedNetwork.Trusted() {
		t.Error("client should be stopped and network not trusted")
	}
}

// TestAgentStartStop tests Start and Stop of Agent.
func TestAgentStartStop(t *testing.T) {
	c := &config.Config{}
	a := NewAgent(c)
	if err := a.Start(); err != nil {
		t.Errorf("could not start agent: %v", err)
	}
	a.Stop()
}

// TestNewAgent tests NewAgent.
func TestNewAgent(t *testing.T) {
	c := &config.Config{}
	a := NewAgent(c)
	if a == nil ||
		a.config == nil ||
		a.dbus == nil ||
		a.ccache == nil ||
		a.krbcfg == nil ||
		a.tnd == nil ||
		a.sleep == nil ||
		a.done == nil ||
		a.closed == nil {

		t.Errorf("got nil, want != nil")
	}
	if c != a.config {
		t.Errorf("got %p, want %p", a.config, c)
	}
}
