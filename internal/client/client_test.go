package client

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/T-Systems-MMS/fw-id-agent/internal/status"
	"github.com/T-Systems-MMS/fw-id-agent/pkg/config"
	krbConfig "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/test/testdata"
)

// initTestServer initializes a test server
func initTestServer(expected string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(expected))
	}))

	return server
}

// getTestCCache returns a credentials cache for testing
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

// TestClientLogin tests login of Client, successful login
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

// TestClientLoginNoResult tests login of Client, successful login but invalid response
func TestClientLoginNoResult(t *testing.T) {
	// create server
	expected := `{ "nonsense": "xyz"}`
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

// TestClientLoginFailed tests login of Client, failed login
func TestClientLoginFailed(t *testing.T) {
	// create server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// TestClientLogout tests logout of client
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
