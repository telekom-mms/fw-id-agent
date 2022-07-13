package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogin(t *testing.T) {
	expected := `{ "keep-alive": 42 }`
	server := initTestServer(expected)
	defer server.Close()

	client := NewClient()
	client.SetURL(server.URL)
	go func() {
		defer close(client.results)
		err := client.login()
		if err != nil {
			t.Errorf("got error from calling login: %v", err)
		}
		r := <-client.Results()
		if !r {
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

	client := NewClient()
	client.SetURL(server.URL)
	go func() {
		defer close(client.results)
		_ = client.login()
		r := <-client.Results()
		if !r {
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

	client := NewClient()
	client.SetURL(server.URL)
	go func() {
		defer close(client.results)
		err := client.login()
		if err == nil {
			t.Errorf("got no error")
		}
		r := <-client.Results()
		if !r {
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

	client := NewClient()
	client.SetURL(server.URL)
	go func() {
		defer close(client.results)
		_ = client.login()
		r := <-client.Results()
		if r {
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
