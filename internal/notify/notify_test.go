package notify

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

// testObj is a dummy D-Bus object for testing.
type testObj struct {
	dbus.Object
	err error
}

func (t *testObj) Call(_ string, _ dbus.Flags, _ ...interface{}) *dbus.Call {
	return &dbus.Call{Err: t.err}
}

// testConn is a dummy D-Bus connection for testing.
type testConn struct {
	err error
}

func (t *testConn) Object(string, dbus.ObjectPath) dbus.BusObject {
	return &testObj{err: t.err}
}

func (t *testConn) Close() error {
	return nil
}

// TestNewNotifier tests NewNotifier
func TestNewNotifier(t *testing.T) {
	// test with no error
	dbusSessionConn = func() (Conn, error) {
		return &testConn{}, nil
	}
	n, err := NewNotifier()
	if err != nil || n == nil || n.conn == nil {
		t.Errorf("error creating notifier")
	}

	// test with error
	dbusSessionConn = func() (Conn, error) {
		return nil, errors.New("test error")
	}
	n, err = NewNotifier()
	if err == nil || n != nil {
		t.Errorf("notifier should not be valid")
	}
}

// TestNotify tests Notify of Notifier.
func TestNotifierNotify(_ *testing.T) {
	// test nil
	var n *Notifier
	n.Notify("test", "this is a test")

	// test with no error
	dbusSessionConn = func() (Conn, error) {
		return &testConn{}, nil
	}
	n, _ = NewNotifier()
	n.Notify("test", "this is a test")
	n.Close()

	// test with error
	dbusSessionConn = func() (Conn, error) {
		return &testConn{err: errors.New("test error")}, nil
	}
	n, _ = NewNotifier()
	n.Notify("test", "this is a test")
	n.Close()
}

// TestNotifierClose tests close of Notify.
func TestNotifierClose(_ *testing.T) {
	// test nil
	var n *Notifier
	n.Close()

	// test normal
	n = &Notifier{conn: &testConn{}}
	n.Close()
}
