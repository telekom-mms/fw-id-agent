package status

import (
	"encoding/json"

	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
)

// KerberosTicket is kerberos ticket info in the agent status
type KerberosTicket struct {
	StartTime int64
	EndTime   int64
}

// Status is the agent status
type Status struct {
	TrustedNetwork bool
	LoggedIn       bool
	Config         *config.Config
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
