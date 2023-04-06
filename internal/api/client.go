package api

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// connectTimeout is the timeout for the client connection attempt
	connectTimeout = 30 * time.Second

	// clientTimeout is the timeout for the entire request/response
	// exchange initiated by the client after a successful connection
	clientTimeout = 30 * time.Second
)

// Client is a Daemon API client
type Client struct {
	sockFile string
}

// Request sends msg to the server and returns the server's response
func (c *Client) Request(msg *Message) *Message {
	// connect to daemon
	conn, err := net.DialTimeout("unix", c.sockFile, connectTimeout)
	if err != nil {
		log.WithError(err).Fatal("Client dial error")
	}
	defer func() {
		_ = conn.Close()
	}()

	// set timeout for entire request/response message exchange
	deadline := time.Now().Add(clientTimeout)
	if err := conn.SetDeadline(deadline); err != nil {
		log.WithError(err).Fatal("Client set deadline error")
	}

	// send message to daemon
	err = WriteMessage(conn, msg)
	if err != nil {
		log.WithError(err).Fatal("Client send message error")
	}

	// receive reply
	reply, err := ReadMessage(conn)
	if err != nil {
		log.WithError(err).Fatal("Client receive message error")
	}

	return reply
}

// Query sends retrieves the status from the server
func (c *Client) Query() []byte {
	// send query to daemon
	msg := NewMessage(TypeQuery, nil)
	reply := c.Request(msg)

	// handle response
	switch reply.Type {
	case TypeOK:
		return reply.Value

	case TypeError:
		log.WithField("error", string(reply.Value)).Error("Client received error reply")
	}
	return nil
}

// NewClient returns a new Client
func NewClient(sockFile string) *Client {
	return &Client{
		sockFile: sockFile,
	}
}
