package status

import (
	"encoding/json"

	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
)

// TrustedNetwork is the current trusted network state
type TrustedNetwork uint32

// TrustedNetwork states
const (
	TrustedNetworkUnknown TrustedNetwork = iota
	TrustedNetworkNotTrusted
	TrustedNetworkTrusted
)

// Trusted returns whether TrustedNetwork is state "trusted"
func (t TrustedNetwork) Trusted() bool {
	return t == TrustedNetworkTrusted
}

// String returns t as string
func (t TrustedNetwork) String() string {
	switch t {
	case TrustedNetworkUnknown:
		return "unknown"
	case TrustedNetworkNotTrusted:
		return "not trusted"
	case TrustedNetworkTrusted:
		return "trusted"
	}
	return ""
}

// LoginState is the current login state
type LoginState uint32

// LoginState states
const (
	LoginStateUnknown LoginState = iota
	LoginStateLoggedOut
	LoginStateLoggingIn
	LoginStateLoggedIn
	LoginStateLoggingOut
)

// LoggedIn returns whether LoginState is state "logged in"
func (l LoginState) LoggedIn() bool {
	return l == LoginStateLoggedIn
}

// String returns l as string
func (l LoginState) String() string {
	switch l {
	case LoginStateUnknown:
		return "unknown"
	case LoginStateLoggedOut:
		return "logged out"
	case LoginStateLoggingIn:
		return "logging in"
	case LoginStateLoggedIn:
		return "logged in"
	case LoginStateLoggingOut:
		return "logging out"
	}
	return ""
}

// KerberosTicket is kerberos ticket info in the agent status
type KerberosTicket struct {
	StartTime int64
	EndTime   int64
}

// Status is the agent status
type Status struct {
	Config         *config.Config
	TrustedNetwork TrustedNetwork
	LoginState     LoginState
	LastKeepAlive  int64
	KerberosTGT    KerberosTicket
}

// JSON returns the Status as JSON
func (s *Status) JSON() ([]byte, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// JSONIndent returns the Status as indented JSON
func (s *Status) JSONIndent() ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}

	return b, nil
}

// NewFromJSON returns a new Status parsed from JSON in b
func NewFromJSON(b []byte) (*Status, error) {
	s := New()
	err := json.Unmarshal(b, s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// New returns a new Status
func New() *Status {
	return &Status{}
}
