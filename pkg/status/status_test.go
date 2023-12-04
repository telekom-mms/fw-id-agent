package status

import (
	"log"
	"reflect"
	"testing"

	"github.com/telekom-mms/fw-id-agent/pkg/config"
)

// TestTrustedNetworkTrusted tests Trusted of TrustedNetwork.
func TestTrustedNetworkTrusted(t *testing.T) {
	for _, f := range []TrustedNetwork{
		TrustedNetworkUnknown,
		TrustedNetworkNotTrusted,
	} {
		if f.Trusted() {
			t.Errorf("%v should not be trusted", f)
		}
	}

	if !TrustedNetworkTrusted.Trusted() {
		t.Error("should be trusted")
	}
}

// TestTrustedNetworkString tests String of TrustedNetwork.
func TestTrustedNetworkString(t *testing.T) {
	for k, v := range map[TrustedNetwork]string{
		TrustedNetworkUnknown:    "unknown",
		TrustedNetworkNotTrusted: "not trusted",
		TrustedNetworkTrusted:    "trusted",
		23:                       "",
	} {
		if k.String() != v {
			t.Errorf("String of %v should return %s", k, v)
		}
	}
}

// TestLoginStateLoggedIn tests LoggedIn of LoginState.
func TestLoginStateLoggedIn(t *testing.T) {
	// test not logged in
	for _, f := range []LoginState{
		LoginStateUnknown,
		LoginStateLoggedOut,
		LoginStateLoggingIn,
		LoginStateLoggingOut,
	} {
		if f.LoggedIn() {
			t.Errorf("LoggedIn of %v should not return true", f)
		}
	}

	// test logged in
	if l := LoginStateLoggedIn; !l.LoggedIn() {
		t.Errorf("LoggedIn of %v should return true", l)
	}
}

// TestLoginStateString tests String of LoginState.
func TestLoginStateString(t *testing.T) {
	for k, v := range map[LoginState]string{
		LoginStateUnknown:    "unknown",
		LoginStateLoggedOut:  "logged out",
		LoginStateLoggingIn:  "logging in",
		LoginStateLoggedIn:   "logged in",
		LoginStateLoggingOut: "logging out",
		23:                   "",
	} {
		if k.String() != v {
			t.Errorf("String of %v should return %s", k, v)
		}
	}
}

// TestKerberosTicketTimesEqual tests TimesEqual of KerberosTicket.
func TestKerberosTicketTimesEqual(t *testing.T) {
	// test not equal
	k := &KerberosTicket{1, 2}
	for _, f := range []*KerberosTicket{
		{0, 0},
		{1, 0},
		{0, 2},
	} {
		if k.TimesEqual(f.StartTime, f.EndTime) {
			t.Errorf("%v and %v should not have equal times", k, f)
		}
	}

	// test equal
	if !k.TimesEqual(1, 2) {
		t.Error("times should be equal")
	}
}

// TestStatusCopy tests Copy of Status.
func TestStatusCopy(t *testing.T) {
	want := &Status{
		Config:         config.Default(),
		TrustedNetwork: TrustedNetworkTrusted,
		LoginState:     LoginStateLoggedIn,
		LastKeepAlive:  2023,
		KerberosTGT: KerberosTicket{
			StartTime: 2023,
			EndTime:   2024,
		},
	}
	got := want.Copy()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestJSON tests JSON and NewFromJSON of Status.
func TestJSON(t *testing.T) {
	s := New()
	b, err := s.JSON()
	if err != nil {
		log.Fatal(err)
	}
	n, err := NewFromJSON(b)
	if err != nil {
		log.Fatal(err)
	}
	if !reflect.DeepEqual(n, s) {
		t.Errorf("got %v, want %v", n, s)
	}
}

// TestJSONIndent tests JSONIndent and NewFromJSON of Status.
func TestJSONIndent(t *testing.T) {
	s := New()
	b, err := s.JSONIndent()
	if err != nil {
		log.Fatal(err)
	}
	n, err := NewFromJSON(b)
	if err != nil {
		log.Fatal(err)
	}
	if !reflect.DeepEqual(n, s) {
		t.Errorf("got %v, want %v", n, s)
	}
}

// TestNewFromJSON tests NewFromJSON.
func TestNewFromJSON(t *testing.T) {
	// test invalid
	if _, err := NewFromJSON([]byte("invalid")); err == nil {
		t.Error("invalid JSON should return error")
	}

	// test valid
	v := New()
	b, err := v.JSON()
	if err != nil {
		log.Fatal(err)
	}
	if n, err := NewFromJSON(b); err != nil {
		t.Error("valid JSON should not return error")
	} else if !reflect.DeepEqual(v, n) {
		t.Errorf("%v and %v should be equal", v, n)
	}
}

// TestNew tests New.
func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Errorf("got nil, want != nil")
	}
}
