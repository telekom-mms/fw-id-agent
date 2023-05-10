package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
	krbConfig "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
)

func TestLogin(t *testing.T) {
	expected := `{ "keep-alive": 42 }`
	server := initTestServer(expected)
	defer server.Close()

	config := &config.Config{ServiceURL: server.URL}
	ccache := &credentials.CCache{}
	krb5conf := krbConfig.New()
	client := NewClient(config, ccache, krb5conf)
	go func() {
		defer close(client.results)
		err := client.login()
		if err != nil {
			t.Errorf("got error from calling login: %v", err)
		}
		r := <-client.Results()
		if !r.LoggedIn() {
			t.Errorf("login result is false")
		}
		if client.keepAlive != 42*time.Minute {
			t.Errorf("keep-alive time not set correctly: %d", client.keepAlive)
		}
	}()
}

func TestLoginNoResult(t *testing.T) {
	expected := `{ "nonsense": "xyz"}`
	server := initTestServer(expected)
	defer server.Close()

	config := &config.Config{ServiceURL: server.URL}
	ccache := &credentials.CCache{}
	krb5conf := krbConfig.New()
	client := NewClient(config, ccache, krb5conf)
	go func() {
		defer close(client.results)
		_ = client.login()
		r := <-client.Results()
		if !r.LoggedIn() {
			t.Errorf("login result is false")
		}
		if client.keepAlive != 5*time.Minute {
			t.Errorf("keep-alive time not set correctly: %d", client.keepAlive)
		}
	}()
}

func TestLoginFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	config := &config.Config{ServiceURL: server.URL}
	ccache := &credentials.CCache{}
	krb5conf := krbConfig.New()
	client := NewClient(config, ccache, krb5conf)
	go func() {
		defer close(client.results)
		err := client.login()
		if err == nil {
			t.Errorf("got no error")
		}
		r := <-client.Results()
		if !r.LoggedIn() {
			t.Errorf("login result is true")
		}
		if client.keepAlive != 5*time.Minute {
			t.Errorf("keep-alive time not set correctly: %d", client.keepAlive)
		}
	}()
}

func TestLogout(t *testing.T) {
	expected := `{}`
	server := initTestServer(expected)
	defer server.Close()

	config := &config.Config{ServiceURL: server.URL}
	ccache := &credentials.CCache{}
	krb5conf := krbConfig.New()
	client := NewClient(config, ccache, krb5conf)
	go func() {
		defer close(client.results)
		_ = client.login()
		r := <-client.Results()
		if r.LoggedIn() {
			t.Errorf("logout result is not false")
		}
	}()
}

func initTestServer(expected string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(expected))
	}))

	return server
}
