// Package dbusapi contains the D-Bus API.
package dbusapi

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	log "github.com/sirupsen/logrus"
)

// D-Bus object path and interface.
const (
	Path      = "/com/telekom_mms/fw_id_agent/Agent"
	Interface = "com.telekom_mms.fw_id_agent.Agent"
)

// Properties.
const (
	PropertyConfig               = "Config"
	PropertyTrustedNetwork       = "TrustedNetwork"
	PropertyLoginState           = "LoginState"
	PropertyLastKeepAliveAt      = "LastKeepAliveAt"
	PropertyKerberosTGTStartTime = "KerberosTGTStartTime"
	PropertyKerberosTGTEndTime   = "KerberosTGTEndTime"
)

// Property "Config" values.
const (
	ConfigInvalid = ""
)

// Property "Trusted Network" states.
const (
	TrustedNetworkUnknown uint32 = iota
	TrustedNetworkNotTrusted
	TrustedNetworkTrusted
)

// Property "Login State" states.
const (
	LoginStateUnknown uint32 = iota
	LoginStateLoggedOut
	LoginStateLoggingIn
	LoginStateLoggedIn
	LoginStateLoggingOut
)

// Property "Last Keep Alive At" values.
const (
	LastKeepAliveAtInvalid int64 = -1
)

// Property "Kerberos TGT Start Time" values.
const (
	KerberosTGTStartTimeInvalid int64 = -1
)

// Property "Kerberos TGT End Time" values.
const (
	KerberosTGTEndTimeInvalid int64 = -1
)

// Methods.
const (
	MethodReLogin = Interface + ".ReLogin"
)

// Request Names.
const (
	RequestReLogin = "ReLogin"
)

// Request is a D-Bus client request.
type Request struct {
	Name       string
	Parameters []any
	Results    []any
	Error      error

	wait chan struct{}
	done chan struct{}
}

// Close completes the request handling.
func (r *Request) Close() {
	close(r.wait)
}

// Wait waits for the completion of request handling.
func (r *Request) Wait() {
	select {
	case <-r.wait:
	case <-r.done:
		r.Error = errors.New("Request aborted")
	}
}

// NewRequest returns a new Request.
func NewRequest(name string, done chan struct{}) *Request {
	return &Request{
		Name: name,
		wait: make(chan struct{}),
		done: done,
	}
}

// agent defines agent interface methods.
type agent struct {
	requests chan *Request
	done     chan struct{}
}

// ReLogin is the "ReLogin" method of the Agent D-Bus interface.
func (a agent) ReLogin(sender dbus.Sender) *dbus.Error {
	log.WithField("sender", sender).Debug("Received D-Bus ReLogin() call")
	request := NewRequest(RequestReLogin, a.done)

	select {
	case a.requests <- request:
	case <-a.done:
		return dbus.NewError(Interface+".ReLoginAborted", []any{"ReLogin aborted"})
	}

	request.Wait()
	if request.Error != nil {
		return dbus.NewError(Interface+".ReLoginAborted", []any{request.Error.Error()})
	}
	return nil
}

// propertyUpdate is an update of a property.
type propertyUpdate struct {
	name  string
	value any
}

// DBusService is the D-Bus Service interface.
type DBusService interface {
	Start() error
	Stop()
	Requests() chan *Request
	SetProperty(name string, value any)
}

// Service is a D-Bus Service.
type Service struct {
	conn     dbusConn
	props    propProperties
	requests chan *Request
	propUps  chan *propertyUpdate
	done     chan struct{}
	closed   chan struct{}
}

// dbusConn is an interface for dbus.Conn to allow for testing.
type dbusConn interface {
	Close() error
	Export(v any, path dbus.ObjectPath, iface string) error
	RequestName(name string, flags dbus.RequestNameFlags) (dbus.RequestNameReply, error)
}

// dbusConnectSessionBus encapsulates dbus.ConnectSessionBus to allow for testing.
var dbusConnectSessionBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
	return dbus.ConnectSessionBus(opts...)
}

// propProperties is an interface for prop.Properties to allow for testing.
type propProperties interface {
	Introspection(iface string) []introspect.Property
	SetMust(iface, property string, v any)
}

// propExport encapsulates prop.Export to allow for testing.
var propExport = func(conn dbusConn, path dbus.ObjectPath, props prop.Map) (propProperties, error) {
	return prop.Export(conn.(*dbus.Conn), path, props)
}

// start starts the service.
func (s *Service) start() {
	defer close(s.closed)
	defer func() { _ = s.conn.Close() }()

	// main loop
	for {
		select {
		case u := <-s.propUps:
			// update property
			log.WithFields(log.Fields{
				"name":  u.name,
				"value": u.value,
			}).Debug("D-Bus updating property")
			s.props.SetMust(Interface, u.name, u.value)

		case <-s.done:
			log.Debug("D-Bus service stopping")
			// set properties values to unknown/invalid to emit
			// properties changed signal and inform clients
			s.props.SetMust(Interface, PropertyConfig, ConfigInvalid)
			s.props.SetMust(Interface, PropertyTrustedNetwork, TrustedNetworkUnknown)
			s.props.SetMust(Interface, PropertyLoginState, LoginStateUnknown)
			s.props.SetMust(Interface, PropertyLastKeepAliveAt, LastKeepAliveAtInvalid)
			s.props.SetMust(Interface, PropertyKerberosTGTStartTime, KerberosTGTStartTimeInvalid)
			s.props.SetMust(Interface, PropertyKerberosTGTEndTime, KerberosTGTEndTimeInvalid)
			return
		}
	}
}

// Start starts the service.
func (s *Service) Start() error {
	// connect to session bus
	conn, err := dbusConnectSessionBus()
	if err != nil {
		return fmt.Errorf("could not connect to D-Bus session bus: %w", err)
	}
	s.conn = conn

	// request name
	reply, err := conn.RequestName(Interface, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("could not request D-Bus name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("requested D-Bus name is already taken")
	}

	// methods
	meths := agent{s.requests, s.done}
	err = conn.Export(meths, Path, Interface)
	if err != nil {
		return fmt.Errorf("could not export D-Bus methods: %w", err)
	}

	// properties
	propsSpec := prop.Map{
		Interface: {
			PropertyConfig: {
				Value:    ConfigInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyTrustedNetwork: {
				Value:    TrustedNetworkUnknown,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyLoginState: {
				Value:    LoginStateUnknown,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyLastKeepAliveAt: {
				Value:    LastKeepAliveAtInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyKerberosTGTStartTime: {
				Value:    KerberosTGTStartTimeInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyKerberosTGTEndTime: {
				Value:    KerberosTGTEndTimeInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
		},
	}
	props, err := propExport(conn, Path, propsSpec)
	if err != nil {
		return fmt.Errorf("could not export D-Bus properties spec: %w", err)
	}
	s.props = props

	// introspection
	n := &introspect.Node{
		Name: Path,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       Interface,
				Methods:    introspect.Methods(meths),
				Properties: props.Introspection(Interface),
			},
		},
	}
	err = conn.Export(introspect.NewIntrospectable(n), Path,
		"org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("could not export D-Bus introspection: %w", err)
	}

	// set properties values to emit properties changed signal and make
	// sure existing clients get updated values after restart
	props.SetMust(Interface, PropertyConfig, ConfigInvalid)
	props.SetMust(Interface, PropertyTrustedNetwork, TrustedNetworkNotTrusted)
	props.SetMust(Interface, PropertyLoginState, LoginStateLoggedOut)
	props.SetMust(Interface, PropertyLastKeepAliveAt, LastKeepAliveAtInvalid)
	props.SetMust(Interface, PropertyKerberosTGTStartTime, KerberosTGTStartTimeInvalid)
	props.SetMust(Interface, PropertyKerberosTGTEndTime, KerberosTGTEndTimeInvalid)

	go s.start()
	return nil
}

// Stop stops the service.
func (s *Service) Stop() {
	close(s.done)
	<-s.closed
}

// Requests returns the requests channel of service.
func (s *Service) Requests() chan *Request {
	return s.requests
}

// SetProperty sets property with name to value.
func (s *Service) SetProperty(name string, value any) {
	select {
	case s.propUps <- &propertyUpdate{name, value}:
	case <-s.done:
	}
}

// NewService returns a new service.
func NewService() *Service {
	return &Service{
		requests: make(chan *Request),
		propUps:  make(chan *propertyUpdate),
		done:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
}
