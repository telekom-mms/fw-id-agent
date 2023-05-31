package status

import (
	"log"
	"reflect"
	"testing"

	"github.com/telekom-mms/fw-id-agent/pkg/config"
)

// TestStatusCopy tests Copy of Status
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

// TestJSON tests JSON and NewFromJSON of Status
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

// TestJSONIndent tests JSONIndent and NewFromJSON of Status
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

// TestNew tests New
func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Errorf("got nil, want != nil")
	}
}
