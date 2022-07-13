package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	krbClient "github.com/jcmturner/gokrb5/v8/client"
	krbConfig "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/spnego"
	log "github.com/sirupsen/logrus"
)

const (
	// retryTimer is the login retry timer in case of errors in seconds
	retryTimer = 15
)

// Client is an identity agent client
type Client struct {
	config    Config
	keepAlive time.Duration
	results   chan bool
	done      chan struct{}
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

// Config is an identity agent client config
type Config struct {
	url     string
	timeout time.Duration
	realm   string
}

// LoginResponse is a login response
type LoginResponse struct {
	KeepAlive int `json:"keep-alive"`
}

func (c *Client) doServiceRequest(api string) (response *http.Response, err error) {
	serviceURL := c.config.url + api
	var currentUser *user.User
	currentUser, err = user.Current()
	if err != nil || currentUser == nil {
		err = fmt.Errorf("%d: error determining user: %w", UserNotSet, err)
		return
	}
	var request *http.Request
	request, err = http.NewRequest("POST", serviceURL, nil)
	if err != nil {
		err = fmt.Errorf("%d: error creating %s request: %w", TokenError, api, err)
		return
	}

	cfg, err := krbConfig.Load("/etc/krb5.conf")
	if err != nil {
		err = fmt.Errorf("%d: could not load KRB5 configuration: %w", TokenError, err)
		return
	}
	cCacheFile, err := getCredentialCacheFilename()
	if err != nil {
		return
	}
	cCache, err := credentials.LoadCCache(cCacheFile)
	if err != nil {
		err = fmt.Errorf("%d: could not load credential cache: %w", TokenError, err)
		return
	}
	krbC, err := krbClient.NewFromCCache(cCache, cfg)
	if err != nil {
		err = fmt.Errorf("%d: could not create KRB5 client: %w", TokenError, err)
		return
	}

	httpClient := http.Client{
		Timeout: c.config.timeout,
		Transport: &http.Transport{
			ResponseHeaderTimeout: c.config.timeout,
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

// createCredentialCacheEnvVar creates an expected environment variable value
// for the credential cache based on the current user ID
func createCredentialCacheEnvVar() string {
	osUser, err := user.Current()
	if err != nil {
		log.WithError(err).
			Error("Agent could not create credential cache environment variable value")
		return ""
	}
	return fmt.Sprintf("FILE:/tmp/krb5cc_%s", osUser.Uid)
}

func getCredentialCacheFilename() (string, error) {
	envVar := os.Getenv("KRB5CCNAME")
	if envVar == "" {
		newEnv := createCredentialCacheEnvVar()
		log.WithField("new", newEnv).
			Debug("Agent could not get environment variable KRB5CCNAME, setting it")
		envVar = newEnv
	}
	if !strings.HasPrefix(envVar, "FILE:") {
		newEnv := createCredentialCacheEnvVar()
		log.WithFields(log.Fields{
			"old": envVar,
			"new": newEnv,
		}).Error("Agent got invalid environment variable KRB5CCNAME, resetting it")
		envVar = newEnv
	}
	if envVar == "" {
		// environment variable still invalid
		return "", fmt.Errorf("%d: environment variable KRB5CCNAME is not set", TokenError)
	}
	return strings.TrimPrefix(envVar, "FILE:"), nil
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
			"default":   c.keepAlive,
		}).Error("Agent received invalid keep alive time at login, using default")
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
				timer.Reset(retryTimer * time.Second)
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

// SetURL sets the identity service url
func (c *Client) SetURL(url string) {
	c.config.url = url
}

// SetRealm sets the identity service url
func (c *Client) SetRealm(realm string) {
	c.config.realm = realm
}

// NewClient returns a new Client
func NewClient() *Client {
	return &Client{
		results:   make(chan bool),
		done:      make(chan struct{}),
		keepAlive: 5 * time.Minute,
		config:    Config{timeout: 30 * time.Second},
	}
}
