package dbusapi

import (
	"errors"
	"reflect"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// TestRequestWaitClose tests Wait and Close of Request.
func TestRequestWaitClose(_ *testing.T) {
	// test closing
	r := Request{
		Name: "test1",
		wait: make(chan struct{}),
		done: make(chan struct{}),
	}
	go func() {
		r.Close()
	}()
	r.Wait()

	// test aborting
	done := make(chan struct{})
	r = Request{
		Name: "test2",
		wait: make(chan struct{}),
		done: done,
	}
	go func() {
		close(done)
	}()
	r.Wait()
}

// TestAgentReLogin tests ReLogin of agent.
func TestAgentReLogin(t *testing.T) {
	// create agent
	requests := make(chan *Request)
	done := make(chan struct{})
	a := agent{
		requests: requests,
		done:     done,
	}

	// run relogin and get results
	want := &Request{
		Name: RequestReLogin,
		done: done,
	}
	got := &Request{}
	go func() {
		r := <-requests
		got = r
		r.Close()
	}()
	err := a.ReLogin("sender")
	if err != nil {
		t.Error(err)
	}

	// check results
	if got.Name != want.Name ||
		!reflect.DeepEqual(got.Parameters, want.Parameters) ||
		!reflect.DeepEqual(got.Results, want.Results) ||
		got.Error != want.Error ||
		got.done != want.done {
		// not equal
		t.Errorf("got %v, want %v", got, want)
	}

	// test with request error
	go func() {
		r := <-requests
		r.Error = errors.New("test error")
		got = r
		r.Close()
	}()
	err = a.ReLogin("sender")
	if err == nil {
		t.Errorf("relogin should return error")
	}

	// test with stopped agent
	close(done)
	err = a.ReLogin("sender")
	if err == nil {
		t.Errorf("relogin should return error")
	}
}

// testConn implements the dbusConn interface for testing.
type testConn struct{}

func (tc *testConn) Close() error {
	return nil
}

func (tc *testConn) Export(any, dbus.ObjectPath, string) error {
	return nil
}

func (tc *testConn) RequestName(string, dbus.RequestNameFlags) (dbus.RequestNameReply, error) {
	return dbus.RequestNameReplyPrimaryOwner, nil
}

// testProperties implements the propProperties interface for testing.
type testProperties struct {
	props map[string]any
}

func (tp *testProperties) Introspection(string) []introspect.Property {
	return nil
}

func (tp *testProperties) SetMust(_, property string, v any) {
	if tp.props == nil {
		// props not set, skip
		return
	}

	// ignore iface, map property to value
	tp.props[property] = v
}

// TestServiceStartStop tests Start and Stop of Service.
func TestServiceStartStop(t *testing.T) {
	dbusConnectSessionBus = func(...dbus.ConnOption) (dbusConn, error) {
		return &testConn{}, nil
	}
	propExport = func(dbusConn, dbus.ObjectPath, prop.Map) (propProperties, error) {
		return &testProperties{}, nil
	}
	s := NewService()
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	s.Stop()
}

// TestServiceRequests tests Requests of Service.
func TestServiceRequests(t *testing.T) {
	s := NewService()
	want := s.requests
	got := s.Requests()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestServiceSetProperty tests SetProperty of Service.
func TestServiceSetProperty(t *testing.T) {
	dbusConnectSessionBus = func(...dbus.ConnOption) (dbusConn, error) {
		return &testConn{}, nil
	}
	properties := &testProperties{props: make(map[string]any)}
	propExport = func(dbusConn, dbus.ObjectPath, prop.Map) (propProperties, error) {
		return properties, nil
	}
	s := NewService()
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	propName := "test-property"
	want := "test-value"

	s.SetProperty(propName, want)
	s.Stop()

	got := properties.props[propName]
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestNewService tests NewService.
func TestNewService(t *testing.T) {
	s := NewService()
	empty := &Service{}
	if reflect.DeepEqual(s, empty) {
		t.Errorf("got empty, want not empty")
	}
}
