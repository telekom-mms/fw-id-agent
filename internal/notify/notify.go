// Package notify contains components for desktop notifications.
package notify

import (
	"math/rand"

	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
)

const (
	iface  = "org.freedesktop.Notifications"
	path   = "/org/freedesktop/Notifications"
	method = "org.freedesktop.Notifications.Notify"
)

// Conn is D-Bus connection.
type Conn interface {
	Object(string, dbus.ObjectPath) dbus.BusObject
	Close() error
}

// Notifier creates desktop notifications.
type Notifier struct {
	conn Conn
	// used to replace notifications with newer ones
	notificationID uint32
}

// dbusSessionConn returns a D-Bus connection.
var dbusSessionConn = func() (Conn, error) {
	return dbus.ConnectSessionBus()
}

// NewNotifier returns a new Notifier.
func NewNotifier() (*Notifier, error) {
	conn, err := dbusSessionConn()
	if err != nil {
		return nil, err
	}
	n := &Notifier{conn: conn, notificationID: rand.Uint32()}
	return n, nil
}

// Notify sends a notification to the user.
func (n *Notifier) Notify(title, message string) {
	if n == nil {
		return
	}
	obj := n.conn.Object(iface, dbus.ObjectPath(path))
	call := obj.Call(method, 0, "", n.notificationID, "", title, message, []string{}, map[string]dbus.Variant{}, int32(5))
	if call.Err != nil {
		log.WithError(call.Err).Error("Agent notify error")
	}
}

// Close closes the Notifier.
func (n *Notifier) Close() {
	if n == nil {
		return
	}
	_ = n.conn.Close()
}
