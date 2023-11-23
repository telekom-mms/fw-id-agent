package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/telekom-mms/fw-id-agent/pkg/status"
)

// TestParseCommandLine tests parseCommandLine.
func TestParseCommandLine(t *testing.T) {
	args := []string{"test", "--version"}
	if err := parseCommandLine(args); err != flag.ErrHelp {
		t.Errorf("unexpected error: %v", err)
	}

	args = []string{"test", "status"}
	if err := parseCommandLine(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	args = []string{"test", "status", "--help"}
	if err := parseCommandLine(args); err != flag.ErrHelp {
		t.Errorf("unexpected error: %v", err)
	}

	args = []string{"test", "monitor"}
	if err := parseCommandLine(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	args = []string{"test", "relogin"}
	if err := parseCommandLine(args); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	args = []string{"test", "invalid-command"}
	if err := parseCommandLine(args); err == nil {
		t.Errorf("should return error")
	}

	args = []string{"test", "--help"}
	if err := parseCommandLine(args); err != flag.ErrHelp {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestPrintStatus tests printStatus.
func TestPrintStatus(t *testing.T) {
	// create status and output buffer
	s := status.New()
	b := &bytes.Buffer{}

	// not verbose
	printStatus(b, s, false)

	got := b.String()
	want := `Trusted Network:    unknown
Login State:        unknown
`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// verbose, timestamps == 0
	b.Reset()
	printStatus(b, s, true)

	got = b.String()
	want = `Trusted Network:    unknown
Login State:        unknown
Last Keep-Alive:
Kerberos TGT:
- Start Time:
- End Time:
Config:             null
`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// verbose, timestamps != 0
	s.LastKeepAlive = 3
	s.KerberosTGT.StartTime = 1
	s.KerberosTGT.EndTime = 2

	b.Reset()
	printStatus(b, s, true)

	got = b.String()
	want = fmt.Sprintf(`Trusted Network:    unknown
Login State:        unknown
Last Keep-Alive:    %s
Kerberos TGT:
- Start Time:       %s
- End Time:         %s
Config:             null
`, time.Unix(3, 0), time.Unix(1, 0), time.Unix(2, 0))

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// testClient is a Client for testing.
type testClient struct {
	status *status.Status
	sub    chan *status.Status
	err    error
}

func (t *testClient) Ping() error                             { return t.err }
func (t *testClient) Query() (*status.Status, error)          { return t.status, t.err }
func (t *testClient) Subscribe() (chan *status.Status, error) { return t.sub, t.err }
func (t *testClient) ReLogin() error                          { return t.err }
func (t *testClient) Close() error                            { return t.err }

// TestRunCommand tests runCommand.
func TestRunCommand(t *testing.T) {
	// create test client
	c := &testClient{}

	// test errors
	c.err = errors.New("test error")

	if err := runCommand(c, "status"); err == nil {
		t.Errorf("command should fail")
	}
	if err := runCommand(c, "monitor"); err == nil {
		t.Errorf("command should fail")
	}
	if err := runCommand(c, "relogin"); err == nil {
		t.Errorf("command should fail")
	}

	// test unknown command
	if err := runCommand(c, "unknown-command"); err != nil {
		t.Errorf("command should not fail")
	}

	// test without errors
	c.err = nil

	// test query
	c.status = status.New()
	if err := runCommand(c, "status"); err != nil {
		t.Errorf("command should not fail")
	}

	// test query with json output
	oldJSON := json
	json = true
	if err := runCommand(c, "status"); err != nil {
		t.Errorf("command should not fail")
	}
	json = oldJSON

	// test monitor
	c.sub = make(chan *status.Status)
	go func() {
		c.sub <- status.New()
		close(c.sub)
	}()
	if err := runCommand(c, "monitor"); err != nil {
		t.Errorf("command should not fail")
	}

	// test relogin
	if err := runCommand(c, "relogin"); err != nil {
		t.Errorf("command should not fail")
	}
}
