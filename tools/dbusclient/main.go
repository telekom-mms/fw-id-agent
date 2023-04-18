package main

import (
	"fmt"

	"github.com/T-Systems-MMS/fw-id-agent/internal/dbusapi"
	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
)

func main() {
	// connect to session bus
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// subscribe to properties changed signals
	if err = conn.AddMatchSignal(
		dbus.WithMatchSender(dbusapi.Interface),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchPathNamespace(dbusapi.Path),
	); err != nil {
		log.Fatal(err)
	}

	// get initial values of properties
	trustedNetwork := dbusapi.TrustedNetworkUnknown
	loginState := dbusapi.LoginStateUnknown
	lastKeepAliveAt := dbusapi.LastKeepAliveAtInvalid
	kerberosTGTStartTime := dbusapi.KerberosTGTStartTimeInvalid
	kerberosTGTEndTime := dbusapi.KerberosTGTEndTimeInvalid

	getProperty := func(name string, val any) {
		err = conn.Object(dbusapi.Interface, dbusapi.Path).
			StoreProperty(dbusapi.Interface+"."+name, val)
		if err != nil {
			log.Fatal(err)
		}
	}
	getProperty(dbusapi.PropertyTrustedNetwork, &trustedNetwork)
	getProperty(dbusapi.PropertyLoginState, &loginState)
	getProperty(dbusapi.PropertyLastKeepAliveAt, &lastKeepAliveAt)
	getProperty(dbusapi.PropertyKerberosTGTStartTime, &kerberosTGTStartTime)
	getProperty(dbusapi.PropertyKerberosTGTEndTime, &kerberosTGTEndTime)

	log.Println("TrustedNetwork:", trustedNetwork)
	log.Println("LoginState:", loginState)
	log.Println("LastKeepAliveAt:", lastKeepAliveAt)
	log.Println("KerberosTGTStartTime:", kerberosTGTStartTime)
	log.Println("KerberosTGTEndTime:", kerberosTGTEndTime)

	// handle signals
	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for s := range c {
		// make sure it's a properties changed signal
		if s.Path != dbusapi.Path ||
			s.Name != "org.freedesktop.DBus.Properties.PropertiesChanged" {
			log.Error("Not a properties changed signal")
			continue
		}

		// check properties changed signal
		if v, ok := s.Body[0].(string); !ok || v != dbusapi.Interface {
			log.Error("Not the right properties changed signal")
			continue
		}

		// get changed properties
		changed, ok := s.Body[1].(map[string]dbus.Variant)
		if !ok {
			log.Error("Invalid changed properties in properties changed signal")
			continue
		}
		for name, value := range changed {
			fmt.Printf("Changed property: %s ", name)
			switch name {
			case dbusapi.PropertyTrustedNetwork:
				if err := value.Store(&trustedNetwork); err != nil {
					log.Fatal(err)
				}
				fmt.Println(trustedNetwork)
			case dbusapi.PropertyLoginState:
				if err := value.Store(&loginState); err != nil {
					log.Fatal(err)
				}
				fmt.Println(loginState)
			case dbusapi.PropertyLastKeepAliveAt:
				if err := value.Store(&lastKeepAliveAt); err != nil {
					log.Fatal(err)
				}
				fmt.Println(lastKeepAliveAt)
			case dbusapi.PropertyKerberosTGTStartTime:
				if err := value.Store(&kerberosTGTStartTime); err != nil {
					log.Fatal(err)
				}
				fmt.Println(kerberosTGTStartTime)
			case dbusapi.PropertyKerberosTGTEndTime:
				if err := value.Store(&kerberosTGTEndTime); err != nil {
					log.Fatal(err)
				}
				fmt.Println(kerberosTGTEndTime)
			}
		}

		// get invalidated properties
		invalid, ok := s.Body[2].([]string)
		if !ok {
			log.Error("Invalid invalidated properties in properties changed signal")
		}
		for _, name := range invalid {
			// not expected to happen currently, but handle it anyway
			switch name {
			case dbusapi.PropertyTrustedNetwork:
				trustedNetwork = dbusapi.TrustedNetworkUnknown
			case dbusapi.PropertyLoginState:
				loginState = dbusapi.LoginStateUnknown
			case dbusapi.PropertyLastKeepAliveAt:
				lastKeepAliveAt = dbusapi.LastKeepAliveAtInvalid
			case dbusapi.PropertyKerberosTGTStartTime:
				kerberosTGTStartTime = dbusapi.KerberosTGTStartTimeInvalid
			case dbusapi.PropertyKerberosTGTEndTime:
				kerberosTGTEndTime = dbusapi.KerberosTGTEndTimeInvalid
			}
			fmt.Printf("Invalidated property: %s\n", name)
		}
	}
}
