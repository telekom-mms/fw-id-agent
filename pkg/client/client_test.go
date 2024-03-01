package client

import (
	"errors"
	"reflect"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/telekom-mms/fw-id-agent/internal/dbusapi"
	"github.com/telekom-mms/fw-id-agent/pkg/status"
)

// TestDBusClientPing tests Ping of DBusClient.
func TestDBusClientPing(t *testing.T) {
	// clean up after tests
	oldPing := ping
	defer func() { ping = oldPing }()

	// test with no error
	client := &DBusClient{}
	ping = func(*DBusClient) error {
		return nil
	}
	err := client.Ping()
	if err != nil {
		t.Errorf("ping returned error %v", err)
	}

	// test with error
	client = &DBusClient{}
	ping = func(*DBusClient) error {
		return errors.New("test error")
	}
	err = client.Ping()
	if err == nil {
		t.Error("ping should return error")
	}
}

// TestUpdateStatusFromProperties tests updateStatusFromProperties. Covers more
// cases than Query tests below.
func TestUpdateStatusFromProperties(t *testing.T) {
	// test invalid
	for _, invalid := range []map[string]dbus.Variant{
		{dbusapi.PropertyConfig: dbus.MakeVariant("invalid")},
		{dbusapi.PropertyConfig: dbus.MakeVariant(0.123)},
		{dbusapi.PropertyTrustedNetwork: dbus.MakeVariant("invalid")},
		{dbusapi.PropertyLoginState: dbus.MakeVariant("invalid")},
		{dbusapi.PropertyLastKeepAliveAt: dbus.MakeVariant("invalid")},
		{dbusapi.PropertyKerberosTGTStartTime: dbus.MakeVariant("invalid")},
		{dbusapi.PropertyKerberosTGTEndTime: dbus.MakeVariant("invalid")},
	} {
		s := status.New()
		err := updateStatusFromProperties(s, invalid)
		if err == nil {
			t.Errorf("should return error for %v", invalid)
		}
	}

	// test valid
	for _, valid := range []map[string]dbus.Variant{
		{},
		{dbusapi.PropertyConfig: dbus.MakeVariant(dbusapi.ConfigInvalid)},
		{dbusapi.PropertyConfig: dbus.MakeVariant("{}")},
		{dbusapi.PropertyTrustedNetwork: dbus.MakeVariant(dbusapi.TrustedNetworkUnknown)},
		{dbusapi.PropertyLoginState: dbus.MakeVariant(dbusapi.LoginStateUnknown)},
		{dbusapi.PropertyLastKeepAliveAt: dbus.MakeVariant(dbusapi.LastKeepAliveAtInvalid)},
		{dbusapi.PropertyKerberosTGTStartTime: dbus.MakeVariant(dbusapi.KerberosTGTStartTimeInvalid)},
		{dbusapi.PropertyKerberosTGTEndTime: dbus.MakeVariant(dbusapi.KerberosTGTEndTimeInvalid)},
	} {
		s := status.New()
		err := updateStatusFromProperties(s, valid)
		if err != nil {
			t.Errorf("should not return error for %v", valid)
		}
	}
}

// TestDBusClientQuery tests Query of DBusClient.
func TestDBusClientQuery(t *testing.T) {
	// clean up after tests
	oldQuery := query
	defer func() { query = oldQuery }()

	// test with no error
	client := &DBusClient{}
	want := status.New()
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}
	got, err := client.Query()
	if err != nil {
		t.Errorf("query returned error %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %p, want %p", got, want)
	}

	// test with query error
	client = &DBusClient{}
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, errors.New("test error")
	}
	_, err = client.Query()
	if err == nil {
		t.Error("query should return error")
	}

	// test with error in properties
	client = &DBusClient{}
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return map[string]dbus.Variant{
			dbusapi.PropertyConfig: dbus.MakeVariant("invalid config"),
		}, nil
	}
	_, err = client.Query()
	if err == nil {
		t.Error("query should return error")
	}
}

// TestHandlePropertiesChanged tests handlePropertiesChanged. Covers more cases
// than Subscribe tests below.
func TestHandlePropertiesChanged(t *testing.T) {
	// test with errors
	for _, invalid := range []*dbus.Signal{
		{},
		{
			Path: dbusapi.Path,
		},
		{
			Path: dbusapi.Path,
			Name: "org.freedesktop.DBus.Properties.PropertiesChanged",
			Body: []any{0},
		},
		{
			Path: dbusapi.Path,
			Name: "org.freedesktop.DBus.Properties.PropertiesChanged",
			Body: []any{dbusapi.Interface, 0},
		},
		{
			Path: dbusapi.Path,
			Name: "org.freedesktop.DBus.Properties.PropertiesChanged",
			Body: []any{dbusapi.Interface, map[string]dbus.Variant{}, 0},
		},
		{
			Path: dbusapi.Path,
			Name: "org.freedesktop.DBus.Properties.PropertiesChanged",
			Body: []any{dbusapi.Interface, map[string]dbus.Variant{
				dbusapi.PropertyTrustedNetwork: dbus.MakeVariant("invalid"),
			}, []string{}},
		},
	} {
		if handlePropertiesChanged(invalid, status.New()) != nil {
			t.Errorf("should return nil for %v", invalid)
		}
	}

	// test with no errors
	valid := &dbus.Signal{
		Path: dbusapi.Path,
		Name: "org.freedesktop.DBus.Properties.PropertiesChanged",
		Body: []any{dbusapi.Interface, map[string]dbus.Variant{
			dbusapi.PropertyConfig:               dbus.MakeVariant(dbusapi.ConfigInvalid),
			dbusapi.PropertyTrustedNetwork:       dbus.MakeVariant(dbusapi.TrustedNetworkUnknown),
			dbusapi.PropertyLoginState:           dbus.MakeVariant(dbusapi.LoginStateUnknown),
			dbusapi.PropertyLastKeepAliveAt:      dbus.MakeVariant(dbusapi.LastKeepAliveAtInvalid),
			dbusapi.PropertyKerberosTGTStartTime: dbus.MakeVariant(dbusapi.KerberosTGTStartTimeInvalid),
			dbusapi.PropertyKerberosTGTEndTime:   dbus.MakeVariant(dbusapi.KerberosTGTEndTimeInvalid),
		}, []string{
			dbusapi.PropertyConfig,
			dbusapi.PropertyTrustedNetwork,
			dbusapi.PropertyLoginState,
			dbusapi.PropertyLastKeepAliveAt,
			dbusapi.PropertyKerberosTGTStartTime,
			dbusapi.PropertyKerberosTGTEndTime,
		}},
	}
	if handlePropertiesChanged(valid, status.New()) == nil {
		t.Errorf("should not return nil for %v", valid)
	}
}

// TestDBusClientSubscribe tests Subscribe of DBusClient.
func TestDBusClientSubscribe(t *testing.T) {
	// clean up after tests
	oldConnectSessionBus := dbusConnectSessionBus
	oldConnSignal := dbusConnSignal
	oldConnAddMatchSignal := dbusConnAddMatchSignal
	oldQuery := query
	defer func() {
		dbusConnectSessionBus = oldConnectSessionBus
		dbusConnSignal = oldConnSignal
		dbusConnAddMatchSignal = oldConnAddMatchSignal
		query = oldQuery
	}()

	// overwrite dbus session bus and conn functions
	dbusConnectSessionBus = func(...dbus.ConnOption) (*dbus.Conn, error) {
		return nil, nil
	}
	dbusConnSignal = func(conn *dbus.Conn, ch chan<- *dbus.Signal) {
		close(ch)
	}

	// set query error
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, errors.New("test error")
	}

	// test with query error
	client, _ := NewDBusClient()
	_, err := client.Subscribe()
	if err == nil {
		t.Error("subscribe should return error")
	}

	// set query ok
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}

	// set match signal error
	dbusConnAddMatchSignal = func(conn *dbus.Conn, options ...dbus.MatchOption) error {
		return errors.New("test error")
	}

	// test with match signal error
	client, _ = NewDBusClient()
	_, err = client.Subscribe()
	if err == nil {
		t.Error("subscribe should return error")
	}

	// set match signal OK
	dbusConnAddMatchSignal = func(conn *dbus.Conn, options ...dbus.MatchOption) error {
		return nil
	}

	// test subscribe and double subscribe
	client, _ = NewDBusClient()

	// first subscribe should be OK
	_, err = client.Subscribe()
	if err != nil {
		t.Errorf("subscribe returned error %v", err)
	}

	// second subscribe should fail
	_, err = client.Subscribe()
	if err == nil {
		t.Error("double subscribe should return error")
	}

	close(client.done)

	// test getting initial status
	client, _ = NewDBusClient()
	c, err := client.Subscribe()
	if err != nil {
		t.Errorf("subscribe returned error %v", err)
	}
	s := <-c
	if !reflect.DeepEqual(s, status.New()) {
		t.Error("status should not be equal to the new status")
	}
	close(client.done)

	// test getting status update
	signals := []*dbus.Signal{
		{},
		{
			Path: dbusapi.Path,
			Name: "org.freedesktop.DBus.Properties.PropertiesChanged",
			Body: []any{dbusapi.Interface, map[string]dbus.Variant{
				dbusapi.PropertyTrustedNetwork: dbus.MakeVariant(dbusapi.TrustedNetworkTrusted),
			}, []string{}},
		},
	}
	dbusConnSignal = func(conn *dbus.Conn, ch chan<- *dbus.Signal) {
		go func() {
			for _, s := range signals {
				ch <- s
			}
			close(ch)
		}()
	}
	client, _ = NewDBusClient()
	c, err = client.Subscribe()
	if err != nil {
		t.Errorf("subscribe returned error %v", err)
	}
	<-c // ignore initial update
	s = <-c
	if s.TrustedNetwork != status.TrustedNetworkTrusted {
		t.Error("network should be trusted")
	}
	close(client.done)

	// test getting status update interrupted
	client, _ = NewDBusClient()
	c, err = client.Subscribe()
	if err != nil {
		t.Errorf("subscribe returned error %v", err)
	}
	<-c // ignore initial update
	close(client.done)
}

// TestDBusClientReLogin tests ReLogin of DBusClient.
func TestDBusClientReLogin(t *testing.T) {
	// clean up after tests
	oldRelogin := relogin
	defer func() {
		relogin = oldRelogin
	}()

	// test with no error
	client := &DBusClient{}
	relogin = func(d *DBusClient) error {
		return nil
	}
	err := client.ReLogin()
	if err != nil {
		t.Errorf("relogin returned error %v", err)
	}

	// test with error
	client = &DBusClient{}
	relogin = func(d *DBusClient) error {
		return errors.New("test error")
	}
	err = client.ReLogin()
	if err == nil {
		t.Error("relogin should return error")
	}
}

// testRWC is a reader writer closer for testing.
type testRWC struct{}

func (t *testRWC) Read([]byte) (int, error)  { return 0, nil }
func (t *testRWC) Write([]byte) (int, error) { return 0, nil }
func (t *testRWC) Close() error              { return nil }

// TestDBusClientClose tests Close of DBusClient.
func TestDBusClientClose(t *testing.T) {
	// test without conn and subscribe
	client := &DBusClient{}
	if err := client.Close(); err != nil {
		t.Error(err)
	}

	// test with conn and subscribe
	conn, err := dbus.NewConn(&testRWC{})
	if err != nil {
		t.Error(err)
	}
	client = &DBusClient{
		conn:       conn,
		subscribed: true,
		updates:    make(chan *status.Status),
		done:       make(chan struct{}),
	}
	close(client.updates)
	if err := client.Close(); err != nil {
		t.Error(err)
	}
}

// TestNewDBusClient tests NewDBusClient.
func TestNewDBusClient(t *testing.T) {
	// clean up after tests
	defer func() { dbusConnectSessionBus = dbus.ConnectSessionBus }()

	// test with errors
	dbusConnectSessionBus = func(...dbus.ConnOption) (*dbus.Conn, error) {
		return nil, errors.New("test error")
	}
	_, err := NewDBusClient()
	if err == nil {
		t.Errorf("connect session bus error should result in error")
	}

	// test without errors
	dbusConnectSessionBus = func(...dbus.ConnOption) (*dbus.Conn, error) {
		return nil, nil
	}
	client, err := NewDBusClient()
	if err != nil {
		t.Error(err)
	}
	if err := client.Close(); err != nil {
		t.Error(err)
	}
}

// TestNewClient tests NewClient.
func TestNewClient(t *testing.T) {
	// clean up after tests
	defer func() { dbusConnectSessionBus = dbus.ConnectSessionBus }()

	// test without errors
	dbusConnectSessionBus = func(...dbus.ConnOption) (*dbus.Conn, error) {
		return nil, nil
	}
	client, err := NewClient()
	if err != nil {
		t.Error(err)
	}
	if err := client.Close(); err != nil {
		t.Error(err)
	}
}
