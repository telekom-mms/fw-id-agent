package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
	krbClient "github.com/jcmturner/gokrb5/v8/client"
	krbConfig "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/spnego"
	log "github.com/sirupsen/logrus"
)

// Client is an identity agent client
type Client struct {
	config    *config.Config
	keepAlive time.Duration
	results   chan bool
	done      chan struct{}

	// current kerberos ccache and config
	// protected by mutex
	mutex    sync.Mutex
	ccache   *credentials.CCache
	krb5conf *krbConfig.Config
}

// Error is an identity agent client error
type Error int8

// Errors
const (
	UserNotSet         Error = 001
	TokenError         Error = 002
	CommunicationError Error = 100
	BackendError       Error = 101
)

// LoginResponse is a login response
type LoginResponse struct {
	KeepAlive int `json:"keep-alive"`
}

func (c *Client) doServiceRequest(api string) (response *http.Response, err error) {
	if c.GetCCache() == nil {
		err = fmt.Errorf("%d: error creating %s request: kerberos CCache not set", TokenError, api)
		return
	}

	if c.GetKrb5Conf() == nil {
		err = fmt.Errorf("%d: error creating %s request: kerberos config not set", TokenError, api)
		return
	}

	serviceURL := c.config.ServiceURL + api
	request, err := http.NewRequest("POST", serviceURL, nil)
	if err != nil {
		err = fmt.Errorf("%d: error creating %s request: %w", TokenError, api, err)
		return
	}

	krbC, err := krbClient.NewFromCCache(c.GetCCache(), c.GetKrb5Conf())
	if err != nil {
		err = fmt.Errorf("%d: could not create KRB5 client: %w", TokenError, err)
		return
	}

	httpClient := http.Client{
		Timeout: c.config.GetTimeout(),
		Transport: &http.Transport{
			ResponseHeaderTimeout: c.config.GetTimeout(),
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	client := spnego.NewClient(krbC, &httpClient, "")

	response, err = client.Do(request)
	if err != nil {
		err = fmt.Errorf("%d: error calling %s request: %w", CommunicationError, api, err)
		return
	}
	if response.StatusCode != 200 {
		buf, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			err = fmt.Errorf("%d: unexpected status code: %d, could not read response: %w", CommunicationError, response.StatusCode, readErr)
			return
		}
		err = fmt.Errorf("%d: unexpected status code: %d, response body: %s", CommunicationError, response.StatusCode, buf)
	}

	return
}

// login sends a login request to the identity service
func (c *Client) login() (err error) {
	response, err := c.doServiceRequest("/login")
	if response != nil {
		defer func() {
			err := response.Body.Close()
			if err != nil {
				log.WithError(err).
					Error("Agent could not close login response body")
			}
		}()
	}
	if err != nil || response.StatusCode != 200 {
		c.results <- false
		return
	}
	var body []byte
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("%d: error reading login response body: %w", BackendError, err)
		c.results <- false
		return
	}
	var responseJSON LoginResponse
	err = json.Unmarshal(body, &responseJSON)
	if err != nil {
		// assume login successful but response has no parseable result
		log.WithError(err).Error("Agent could not parse login response")
		c.results <- true
		return
	}
	if responseJSON.KeepAlive > 0 {
		c.keepAlive = time.Duration(responseJSON.KeepAlive) * time.Minute
	} else {
		log.WithFields(log.Fields{
			"keepAlive": responseJSON.KeepAlive,
			"current":   c.keepAlive,
			"default":   c.config.KeepAlive,
		}).Error("Agent received invalid keep alive time at login, using current")
	}

	c.results <- true
	return
}

func (c *Client) logout() (err error) {
	_, err = c.doServiceRequest("/logout")
	c.results <- false
	return
}

// start starts executing the client
func (c *Client) start() {
	defer close(c.results)

	timer := time.NewTimer(0)
	for {
		select {
		case <-timer.C:
			err := c.login()
			if err != nil {
				// error during login attempt, log error and
				// reset timer to retry timer value
				log.WithError(err).Error("Agent got error during method login")
				timer.Reset(c.config.GetRetryTimer())
				break
			}
			timer.Reset(c.keepAlive)

		case <-c.done:
			err := c.logout()
			if err != nil {
				// error during logout attempt, this might be
				// caused by not being connected to the trusted
				// network anymore, so only log with debug
				// level to avoid user confusion
				log.WithError(err).Debug("Agent got error during method logout")
			}
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

// Start starts the client {
func (c *Client) Start() {
	go c.start()
}

// Stop stops the client
func (c *Client) Stop() {
	close(c.done)
	for range c.results {
		// wait for result channel close
	}
}

// Results returns the result channel
func (c *Client) Results() chan bool {
	return c.results
}

// SetCCache sets the kerberos CCache in the client
func (c *Client) SetCCache(ccache *credentials.CCache) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.ccache = ccache
}

// GetCCache returns the kerberos CCache in the client
func (c *Client) GetCCache() *credentials.CCache {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.ccache
}

// SetKrb5Conf sets the kerberos config in the client
func (c *Client) SetKrb5Conf(conf *krbConfig.Config) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.krb5conf = conf
}

// GetKrb5Conf returns the kerberos config in the client
func (c *Client) GetKrb5Conf() *krbConfig.Config {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.krb5conf
}

// NewClient returns a new Client
func NewClient(config *config.Config, ccache *credentials.CCache, krb5conf *krbConfig.Config) *Client {
	return &Client{
		results:   make(chan bool),
		done:      make(chan struct{}),
		keepAlive: config.GetKeepAlive(),
		config:    config,
		ccache:    ccache,
		krb5conf:  krb5conf,
	}
}
