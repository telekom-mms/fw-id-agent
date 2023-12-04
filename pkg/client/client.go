// Package client contains the FW-ID-Agent client.
package client

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/telekom-mms/fw-id-agent/internal/dbusapi"
	"github.com/telekom-mms/fw-id-agent/pkg/config"
	"github.com/telekom-mms/fw-id-agent/pkg/status"
)

// Client is a FW-ID-Agent client.
type Client interface {
	Ping() error
	Query() (*status.Status, error)
	Subscribe() (chan *status.Status, error)
	ReLogin() error
	Close() error
}

// DBusClient is a FW-ID-Agent client that uses the D-Bus API of FW-ID-Agent.
type DBusClient struct {
	mutex sync.Mutex

	// conn is the D-Bus connection
	conn *dbus.Conn

	// subscribed specifies whether the client is subscribed to
	// PropertiesChanged D-Bus signals
	subscribed bool

	// update is used for status updates
	updates chan *status.Status

	// done signals termination of the client
	done chan struct{}
}

// dbusConnectSessionBus is dbus.ConnectSystemBus for testing.
var dbusConnectSessionBus = dbus.ConnectSessionBus

// updateStatusFromProperties updates status s from D-Bus properties in props.
func updateStatusFromProperties(s *status.Status, props map[string]dbus.Variant) error {
	// create a temporary status, try to set all values in temporary
	// status, if we received valid properties (no type conversion or JSON
	// parsing errors) set real status
	temp := status.New()
	for _, dest := range []*status.Status{temp, s} {
		for k, v := range props {
			var err error
			switch k {
			case dbusapi.PropertyConfig:
				s := dbusapi.ConfigInvalid
				if err := v.Store(&s); err != nil {
					return err
				}
				if s == dbusapi.ConfigInvalid {
					dest.Config = nil
				} else {
					c, err := config.NewFromJSON([]byte(s))
					if err != nil {
						return err
					}
					dest.Config = c
				}
			case dbusapi.PropertyTrustedNetwork:
				err = v.Store(&dest.TrustedNetwork)
			case dbusapi.PropertyLoginState:
				err = v.Store(&dest.LoginState)
			case dbusapi.PropertyLastKeepAliveAt:
				err = v.Store(&dest.LastKeepAlive)
			case dbusapi.PropertyKerberosTGTStartTime:
				err = v.Store(&dest.KerberosTGT.StartTime)
			case dbusapi.PropertyKerberosTGTEndTime:
				err = v.Store(&dest.KerberosTGT.EndTime)
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ping calls the ping method to check if FW-ID-Agent is running.
var ping = func(d *DBusClient) error {
	return d.conn.Object(dbusapi.Interface, dbusapi.Path).
		Call("org.freedesktop.DBus.Peer.Ping", 0).Err
}

// Ping pings the FW-ID-Agent to check if it is running.
func (d *DBusClient) Ping() error {
	return ping(d)
}

// query retrieves the D-Bus properties from the agent.
var query = func(d *DBusClient) (map[string]dbus.Variant, error) {
	// get all properties
	props := make(map[string]dbus.Variant)
	if err := d.conn.Object(dbusapi.Interface, dbusapi.Path).
		Call("org.freedesktop.DBus.Properties.GetAll", 0, dbusapi.Interface).
		Store(props); err != nil {
		return nil, err
	}

	// return properties
	return props, nil
}

// Query retrieves the status.
func (d *DBusClient) Query() (*status.Status, error) {
	// get properties
	props, err := query(d)
	if err != nil {
		return nil, err
	}

	// get status from properties
	status := status.New()
	if err := updateStatusFromProperties(status, props); err != nil {
		return nil, err
	}

	// return current status
	return status, nil
}

// handlePropertiesChanged handles a PropertiesChanged D-Bus signal.
func handlePropertiesChanged(s *dbus.Signal, stat *status.Status) *status.Status {
	// make sure it's a properties changed signal
	if s.Path != dbusapi.Path ||
		s.Name != "org.freedesktop.DBus.Properties.PropertiesChanged" {
		return nil
	}

	// check properties changed signal
	if v, ok := s.Body[0].(string); !ok || v != dbusapi.Interface {
		return nil
	}

	// get changed properties, update current status
	changed, ok := s.Body[1].(map[string]dbus.Variant)
	if !ok {
		return nil
	}

	err := updateStatusFromProperties(stat, changed)
	if err != nil {
		return nil
	}

	// get invalidated properties
	invalid, ok := s.Body[2].([]string)
	if !ok {
		return nil
	}
	for _, name := range invalid {
		// not expected to happen currently, but handle it anyway
		switch name {
		case dbusapi.PropertyConfig:
			stat.Config = nil
		case dbusapi.PropertyTrustedNetwork:
			stat.TrustedNetwork = status.TrustedNetworkUnknown
		case dbusapi.PropertyLoginState:
			stat.LoginState = status.LoginStateUnknown
		case dbusapi.PropertyLastKeepAliveAt:
			stat.LastKeepAlive = dbusapi.LastKeepAliveAtInvalid
		case dbusapi.PropertyKerberosTGTStartTime:
			stat.KerberosTGT.StartTime = dbusapi.KerberosTGTStartTimeInvalid
		case dbusapi.PropertyKerberosTGTEndTime:
			stat.KerberosTGT.EndTime = dbusapi.KerberosTGTEndTimeInvalid
		}
	}

	return stat
}

// setSubscribed tries to set subscribed to true and returns true if successful.
func (d *DBusClient) setSubscribed() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.subscribed {
		// already subscribed
		return false
	}
	d.subscribed = true
	return true
}

// isSubscribed returns whether subscribed is set.
func (d *DBusClient) isSubscribed() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.subscribed
}

// dbusConnAddMatchSignal calls AddMatchSignal on dbus.Conn.
var dbusConnAddMatchSignal = func(conn *dbus.Conn, options ...dbus.MatchOption) error {
	return conn.AddMatchSignal(options...)
}

// dbusConnSignal calls Signal on dbus.Conn.
var dbusConnSignal = func(conn *dbus.Conn, ch chan<- *dbus.Signal) {
	conn.Signal(ch)
}

// Subscribe subscribes to PropertiesChanged D-Bus signals, converts incoming
// PropertiesChanged signals to status updates and sends those updates over the
// returned channel.
func (d *DBusClient) Subscribe() (chan *status.Status, error) {
	// make sure this only runs once
	if ok := d.setSubscribed(); !ok {
		return nil, fmt.Errorf("already subscribed")
	}

	// query current status to get initial values
	status, err := d.Query()
	if err != nil {
		return nil, err
	}

	// subscribe to properties changed signals
	if err := dbusConnAddMatchSignal(
		d.conn,
		dbus.WithMatchSender(dbusapi.Interface),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchPathNamespace(dbusapi.Path),
	); err != nil {
		return nil, err
	}

	// handle signals
	c := make(chan *dbus.Signal, 10)
	dbusConnSignal(d.conn, c)

	// handle properties
	go func() {
		defer close(d.updates)

		// send initial status
		select {
		case d.updates <- status.Copy():
		case <-d.done:
			return
		}

		// handle signals
		for s := range c {
			// get status update from signal
			update := handlePropertiesChanged(s, status.Copy())
			if update == nil {
				// invalid update
				continue
			}

			// valid update, save it as current status
			status = update.Copy()

			// send status update
			select {
			case d.updates <- update:
			case <-d.done:
				return
			}
		}
	}()

	return d.updates, nil
}

// relogin sends a re-login request to the agent.
var relogin = func(d *DBusClient) error {
	return d.conn.Object(dbusapi.Interface, dbusapi.Path).
		Call(dbusapi.MethodReLogin, 0).Store()
}

// ReLogin sends a re-login request to the agent.
func (d *DBusClient) ReLogin() error {
	return relogin(d)
}

// Close closes the DBusClient.
func (d *DBusClient) Close() error {
	var err error

	if d.conn != nil {
		err = d.conn.Close()
	}

	if d.isSubscribed() {
		close(d.done)
		for range d.updates {
			// wait for channel close
		}
	}

	return err
}

// NewDBusClient returns a new DBusClient.
func NewDBusClient() (*DBusClient, error) {
	// connect to session bus
	conn, err := dbusConnectSessionBus()
	if err != nil {
		return nil, err
	}

	// create client
	client := &DBusClient{
		conn:    conn,
		updates: make(chan *status.Status),
		done:    make(chan struct{}),
	}

	return client, nil
}

// NewClient returns a new Client.
func NewClient() (Client, error) {
	return NewDBusClient()
}
