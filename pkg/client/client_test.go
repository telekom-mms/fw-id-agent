package client

import (
	"reflect"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/telekom-mms/fw-id-agent/pkg/status"
)

// TestDBusClientQuery tests Query of DBusClient
func TestDBusClientQuery(t *testing.T) {
	client := &DBusClient{}
	want := status.New()
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}
	got, err := client.Query()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestDBusClientReLogin tests ReLogin of DBusClient
func TestDBusClientReLogin(t *testing.T) {
	client := &DBusClient{}
	relogin = func(d *DBusClient) error {
		return nil
	}
	err := client.ReLogin()
	if err != nil {
		t.Error(err)
	}
}

// TestNewDBusClient tests NewDBusClient
func TestNewDBusClient(t *testing.T) {
	dbusConnectSessionBus = func() (*dbus.Conn, error) {
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

// TestNewClient tests NewClient
func TestNewClient(t *testing.T) {
	dbusConnectSessionBus = func() (*dbus.Conn, error) {
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
