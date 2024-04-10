package client

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/iotest"
	"time"

	krbConfig "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/spnego"
	"github.com/jcmturner/gokrb5/v8/test/testdata"
	"github.com/telekom-mms/fw-id-agent/pkg/config"
	"github.com/telekom-mms/fw-id-agent/pkg/status"
)

// initTestServer initializes a test server.
func initTestServer(expected string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(expected))
	}))

	return server
}

// getTestCCache returns a credentials cache for testing.
func getTestCCache(t *testing.T) *credentials.CCache {
	b, err := hex.DecodeString(testdata.CCACHE_TEST)
	if err != nil {
		t.Fatal("Error decoding test data")
	}
	ccache := &credentials.CCache{}
	err = ccache.Unmarshal(b)
	if err != nil {
		t.Fatalf("Error parsing cache: %v", err)
	}
	return ccache
}

// TestClientDoServiceRequestErrors tests doServiceRequest of Client, errors.
func TestClientDoServiceRequestErrors(t *testing.T) {
	t.Run("invalid ccache", func(t *testing.T) {
		config := config.Default()
		krb5conf := krbConfig.New()
		client := NewClient(config, nil, krb5conf)
		if _, err := client.doServiceRequest("", time.Hour); err == nil {
			t.Error("service request should fail")
		}
	})

	t.Run("invalid krb5 config", func(t *testing.T) {
		config := config.Default()
		ccache := getTestCCache(t)
		client := NewClient(config, ccache, nil)
		if _, err := client.doServiceRequest("", time.Hour); err == nil {
			t.Error("service request should fail")
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		httpNewRequest = func(string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("test error")
		}
		defer func() { httpNewRequest = http.NewRequest }()

		config := config.Default()
		ccache := getTestCCache(t)
		krb5conf := krbConfig.New()
		client := NewClient(config, ccache, krb5conf)
		if _, err := client.doServiceRequest("", time.Hour); err == nil {
			t.Error("service request should fail")
		}
	})

	t.Run("request error", func(t *testing.T) {
		config := config.Default()
		ccache := getTestCCache(t)
		krb5conf := krbConfig.New()
		client := NewClient(config, ccache, krb5conf)
		if _, err := client.doServiceRequest("", time.Hour); err == nil {
			t.Error("service request should fail")
		}
	})

	t.Run("error response", func(t *testing.T) {
		old := clientDo
		clientDo = func(*spnego.Client, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 300,
				Body:       io.NopCloser(iotest.ErrReader(errors.New("test error"))),
			}, nil
		}
		defer func() { clientDo = old }()

		config := config.Default()
		ccache := getTestCCache(t)
		krb5conf := krbConfig.New()
		client := NewClient(config, ccache, krb5conf)
		if _, err := client.doServiceRequest("", time.Hour); err == nil {
			t.Error("service request should fail")
		}
	})
}

// TestClientLogin tests login of Client, successful login.
func TestClientLogin(t *testing.T) {
	// create server
	expected := `{ "keep-alive": 42 }`
	server := initTestServer(expected)
	defer server.Close()

	// create config
	config := config.Default()
	config.ServiceURL = server.URL

	// create and run client
	ccache := getTestCCache(t)
	krb5conf := krbConfig.New()
	client := NewClient(config, ccache, krb5conf)

	go func() {
		defer close(client.results)
		err := client.login()
		if err != nil {
			t.Errorf("got error from calling login: %v", err)
		}
	}()

	// check "logging in"
	r := <-client.Results()
	if r != status.LoginStateLoggingIn {
		t.Errorf("client not logging in")
	}

	// check "logged in" and keep-alive time
	r = <-client.Results()
	if r != status.LoginStateLoggedIn {
		t.Errorf("client not logged in")
	}
	if client.keepAlive != 42*time.Minute {
		t.Errorf("keep-alive time not set correctly: %d", client.keepAlive)
	}
}

// TestClientLoginNoResult tests login of Client, successful login but invalid response.
func TestClientLoginNoResult(t *testing.T) {
	// create server with invalid responses:
	// - no keep-alive
	// - empty
	for _, server := range []*httptest.Server{
		initTestServer(`{ "nonsense": "xyz"}`),
		initTestServer(``),
	} {
		defer server.Close()

		// create config
		config := config.Default()
		config.ServiceURL = server.URL

		// create and run client
		ccache := getTestCCache(t)
		krb5conf := krbConfig.New()
		client := NewClient(config, ccache, krb5conf)

		go func() {
			defer close(client.results)
			_ = client.login()
		}()

		// check "logging in"
		r := <-client.Results()
		if r != status.LoginStateLoggingIn {
			t.Errorf("client not logging in")
		}

		// check "logged in" and keep-alive
		r = <-client.Results()
		if r != status.LoginStateLoggedIn {
			t.Errorf("client not logged in")
		}
		if client.keepAlive != 5*time.Minute {
			t.Errorf("keep-alive time not set correctly: %d", client.keepAlive)
		}
	}
}

// TestClientLoginFailed tests login of Client, failed login.
func TestClientLoginFailed(t *testing.T) {
	// restore clientDo after the tests
	old := clientDo
	defer func() { clientDo = old }()

	// run tests:
	// - 404
	// - 200 with read error
	for _, response := range []*http.Response{
		{StatusCode: 404, Body: io.NopCloser(&bytes.Buffer{})},
		{StatusCode: 200, Body: io.NopCloser(iotest.ErrReader(errors.New("test error")))},
	} {
		clientDo = func(*spnego.Client, *http.Request) (*http.Response, error) {

			return response, nil
		}

		// create config
		config := config.Default()

		// create and run client
		ccache := getTestCCache(t)
		krb5conf := krbConfig.New()
		client := NewClient(config, ccache, krb5conf)

		go func() {
			defer close(client.results)
			err := client.login()
			if err == nil {
				t.Errorf("got no error")
			}
		}()

		// check "logging in"
		r := <-client.Results()
		if r != status.LoginStateLoggingIn {
			t.Errorf("client not logging in")
		}

		// check "logged out" and keep-alive
		r = <-client.Results()
		if r != status.LoginStateLoggedOut {
			t.Errorf("client not logged out")
		}
		if client.keepAlive != 5*time.Minute {
			t.Errorf("keep-alive time not set correctly: %d", client.keepAlive)
		}
	}
}

// TestClientLogout tests logout of Client.
func TestClientLogout(t *testing.T) {
	// create server
	expected := `{}`
	server := initTestServer(expected)
	defer server.Close()

	// create config
	config := config.Default()
	config.ServiceURL = server.URL

	// create and run client
	ccache := &credentials.CCache{}
	krb5conf := krbConfig.New()
	client := NewClient(config, ccache, krb5conf)

	go func() {
		defer close(client.results)
		_ = client.logout()
	}()

	// check "logging out"
	r := <-client.Results()
	if r != status.LoginStateLoggingOut {
		t.Errorf("client not logging out")
	}

	// check "logged out"
	r = <-client.Results()
	if r != status.LoginStateLoggedOut {
		t.Errorf("client not logged out")
	}
}

// TestClientStartStop tests Start and Stop of Client.
func TestClientStartStop(t *testing.T) {
	t.Run("failed login", func(t *testing.T) {
		// create server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(404)
		}))
		defer server.Close()

		// create config
		config := config.Default()
		config.ServiceURL = server.URL

		// create and run client
		ccache := getTestCCache(t)
		krb5conf := krbConfig.New()
		client := NewClient(config, ccache, krb5conf)
		client.Start()

		// check "logging in"
		r := <-client.Results()
		if r != status.LoginStateLoggingIn {
			t.Errorf("client not logging in")
		}

		// check "logged out"
		r = <-client.Results()
		if r != status.LoginStateLoggedOut {
			t.Errorf("client not logged out")
		}

		client.Stop()
	})

	t.Run("successful login", func(t *testing.T) {
		// create server
		expected := `{ "keep-alive": 42 }`
		server := initTestServer(expected)
		defer server.Close()

		// create config
		config := config.Default()
		config.ServiceURL = server.URL

		// create and run client
		ccache := getTestCCache(t)
		krb5conf := krbConfig.New()
		client := NewClient(config, ccache, krb5conf)
		client.Start()

		// check "logging in"
		r := <-client.Results()
		if r != status.LoginStateLoggingIn {
			t.Errorf("client not logging in")
		}

		// check "logged in"
		r = <-client.Results()
		if r != status.LoginStateLoggedIn {
			t.Errorf("client not logged in")
		}

		client.Stop()
	})

	t.Run("immediate stop without consumer", func(t *testing.T) {
		// create config
		config := config.Default()

		// create and run client
		ccache := getTestCCache(t)
		krb5conf := krbConfig.New()
		client := NewClient(config, ccache, krb5conf)
		client.Start()
		client.Stop()
	})
}

// TestClientSetGetCCache tests SetCCache and GetCCache of Client.
func TestClientSetGetCCache(t *testing.T) {
	client := NewClient(config.Default(), nil, nil)
	want := getTestCCache(t)
	client.SetCCache(want)
	got := client.GetCCache()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestClientSetGetKrb5Conf tests SetKrb5Conf and GetKrb5Conf of Client.
func TestClientSetGetKrb5Conf(t *testing.T) {
	client := NewClient(config.Default(), nil, nil)
	want := krbConfig.New()
	client.SetKrb5Conf(want)
	got := client.GetKrb5Conf()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewClient tests NewClient.
func TestNewClient(t *testing.T) {
	config := config.Default()
	ccache := getTestCCache(t)
	krb5conf := krbConfig.New()
	client := NewClient(config, ccache, krb5conf)
	if client == nil ||
		client.results == nil ||
		client.done == nil ||
		client.closed == nil ||
		client.config != config ||
		client.ccache != ccache ||
		client.krb5conf != krb5conf {
		t.Error("invalid client")
	}
}
