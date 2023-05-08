package agent

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/T-Systems-MMS/fw-id-agent/internal/config"
	log "github.com/sirupsen/logrus"
)

var (
	// Version is the agent version, to be set at compile time
	Version = "unknown"
)

// command line argument names
const (
	argConfig     = "config"
	argVersion    = "version"
	argServiceURL = "serviceurl"
	argRealm      = "realm"
	argKeepAlive  = "keepalive"
	argTimeout    = "timeout"
	argRetryTimer = "retrytimer"
	argTNDServers = "tndservers"
	argVerbose    = "verbose"
	argMinUserID  = "minuserid"
	argStartDelay = "startdelay"
)

// flagIsSet returns whether flag with name is set as command line argument
func flagIsSet(name string) bool {
	isSet := false
	flag.Visit(func(f *flag.Flag) {
		if name == f.Name {
			isSet = true
		}
	})
	return isSet
}

// parseTNDServers parses the TND servers command line argument
func parseTNDServers(servers string) ([]config.TNDHTTPSConfig, bool) {
	if servers == "" {
		return nil, false
	}
	list := []config.TNDHTTPSConfig{}
	for _, s := range strings.Split(servers, ",") {
		i := strings.LastIndex(s, ":")
		if i == -1 || len(s) < i+2 {
			return nil, false
		}
		url := s[:i]
		hash := strings.ToLower(s[i+1:])
		server := config.TNDHTTPSConfig{URL: url, Hash: hash}
		list = append(list, server)
	}
	return list, true
}

// Run is the main entry point
func Run() {
	// parse command line arguments
	defaults := config.Default()
	cfgFile := flag.String(argConfig, "", "Set config `file`")
	ver := flag.Bool(argVersion, false, "print version")
	serviceURL := flag.String(argServiceURL, "", "Set service URL")
	realm := flag.String(argRealm, "", "Set kerberos realm")
	keepAlive := flag.Int(argKeepAlive, defaults.KeepAlive, "Set default client keep-alive in `minutes`")
	timeout := flag.Int(argTimeout, defaults.Timeout, "Set client request timeout in `seconds`")
	retryTimer := flag.Int(argRetryTimer, defaults.RetryTimer, "Set client login retry timer in case of errors in `seconds`")
	tndServers := flag.String(argTNDServers, "", "Set comma-separated `list` of TND server url:hash pairs")
	verbose := flag.Bool(argVerbose, defaults.Verbose, "Set verbose output")
	minUserID := flag.Int(argMinUserID, defaults.MinUserID, "Set minimum allowed user `ID`")
	startDelay := flag.Int(argStartDelay, defaults.StartDelay, "Set agent start delay in `seconds`")
	flag.Parse()

	// print version?
	if *ver {
		fmt.Println(Version)
		os.Exit(0)
	}

	// load config or try defaults
	cfg := config.Default()
	if flagIsSet("config") {
		c, err := config.Load(*cfgFile)
		if err != nil {
			log.WithError(err).Fatal("Agent could not load config")
		}
		cfg = c
	}

	// overwrite config settings with command line arguments
	if flagIsSet(argServiceURL) {
		cfg.ServiceURL = *serviceURL
	}
	if flagIsSet(argRealm) {
		cfg.Realm = *realm
	}
	if flagIsSet(argKeepAlive) {
		cfg.KeepAlive = *keepAlive
	}
	if flagIsSet(argTimeout) {
		cfg.Timeout = *timeout
	}
	if flagIsSet(argRetryTimer) {
		cfg.RetryTimer = *retryTimer
	}
	if flagIsSet(argTNDServers) {
		servers, ok := parseTNDServers(*tndServers)
		if !ok {
			log.WithField(argTNDServers, *tndServers).Fatal("Agent could not parse TND servers")
		}
		cfg.TND.HTTPSServers = servers
	}
	if flagIsSet(argVerbose) {
		cfg.Verbose = *verbose
	}
	if flagIsSet(argMinUserID) {
		cfg.MinUserID = *minUserID
	}
	if flagIsSet(argStartDelay) {
		cfg.StartDelay = *startDelay
	}

	// check if config is valid
	if !cfg.Valid() {
		log.Fatal("Agent could not get valid config from file or command line arguments")
	}

	// set verbose output
	if cfg.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	log.WithField("config", cfg).Debug("Agent starting with valid config")

	// check user
	osUser, err := user.Current()
	if err != nil {
		log.WithError(err).Fatal("Agent could not get current user")
	}
	uid, err := strconv.Atoi(osUser.Uid)
	if err != nil {
		log.WithError(err).Fatal("Agent started with invalid user id")
	}
	if uid < cfg.MinUserID {
		log.Fatal("Agent started with user id lower than minimum allowed user id")
	}

	// give the user's desktop environment some time to start after login,
	// so we do not miss notifications
	log.WithField("seconds", cfg.StartDelay).Debug("Agent sleeping before starting")
	time.Sleep(cfg.GetStartDelay())

	// start agent
	log.Debug("Agent starting")
	a := NewAgent(cfg)
	a.Start()

	// catch interrupt and clean up
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	a.Stop()
}
