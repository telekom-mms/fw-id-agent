package api

import (
	"log"
	"reflect"
	"testing"
)

// initTestClientServer returns a client and server for testing;
// the server simply closes client requests
func initTestClientServer() (*Client, *Server) {
	server := NewServer("test.sock")
	client := NewClient(server.sockFile)
	go func() {
		for r := range server.requests {
			log.Println(r)
			r.Close()
		}
	}()
	return client, server
}

// TestClientRequest tests Request of Client
func TestClientRequest(t *testing.T) {
	client, server := initTestClientServer()
	server.Start()
	reply := client.Request(NewMessage(TypeQuery, nil))
	server.Stop()

	log.Println(reply)
}

// TestClientQuery tests Query of Client
func TestClientQuery(t *testing.T) {
	server := NewServer("test.sock")
	client := NewClient(server.sockFile)
	status := []byte("some status")
	go func() {
		for r := range server.requests {
			// handle query requests only,
			// reply with status
			log.Println(r)
			r.Reply(status)
			r.Close()
		}
	}()
	server.Start()
	want := status
	got := client.Query()
	server.Stop()

	log.Println(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewClient tests NewClient
func TestNewClient(t *testing.T) {
	sockFile := "test.sock"
	client := NewClient(sockFile)
	got := client.sockFile
	want := sockFile
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
