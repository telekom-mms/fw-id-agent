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

type Conn interface {
	Object(string, dbus.ObjectPath) dbus.BusObject
	Close() error
}

type Notifier struct {
	conn Conn
	// used to replace notifications with newer ones
	notificationId uint32
}

func dbusSessionConn() (*dbus.Conn, error) {
	return dbus.ConnectSessionBus()
}

func NewNotifier() *Notifier {
	conn, err := dbusSessionConn()
	if err != nil {
		log.WithError(err).Fatal("Could not connect to D-Bus session bus")
	}
	n := &Notifier{conn: conn, notificationId: rand.Uint32()}
	return n
}

// Notify sends a notification to the user
func (n *Notifier) Notify(title, message string) {
	obj := n.conn.Object(iface, dbus.ObjectPath(path))
	call := obj.Call(method, 0, "", n.notificationId, "", title, message, []string{}, map[string]dbus.Variant{}, int32(5))
	if call.Err != nil {
		log.WithError(call.Err).Error("Agent notify error")
	}
}

func (n *Notifier) Close() {
	_ = n.conn.Close()
}
