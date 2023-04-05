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
	// version is the agent version, to be set at compile time
	version = "unknown"
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
	cfgFile := flag.String("config", "config.json", "Set config `file`")
	ver := flag.Bool("version", false, "print version")
	serviceURL := flag.String("serviceurl", "", "Set service URL")
	realm := flag.String("realm", "", "Set kerberos realm")
	keepAlive := flag.Int("keepalive", defaults.KeepAlive, "Set default client keep-alive in `minutes`")
	timeout := flag.Int("timeout", defaults.Timeout, "Set client request timeout in `seconds`")
	retryTimer := flag.Int("retrytimer", defaults.RetryTimer, "Set client login retry timer in case of errors in `seconds`")
	tndServers := flag.String("tndservers", "", "Set comma-separated `list` of TND server url:hash pairs")
	verbose := flag.Bool("verbose", defaults.Verbose, "Set verbose output")
	minUserID := flag.Int("minuserid", defaults.MinUserID, "Set minimum allowed user `ID`")
	startDelay := flag.Int("startdelay", defaults.StartDelay, "Set agent start delay in `seconds`")
	flag.Parse()

	// print version?
	if *ver {
		fmt.Println(version)
		os.Exit(0)
	}

	// load config
	cfg, err := config.Load(*cfgFile)
	if err != nil {
		log.WithError(err).Fatal("Agent could not load config")
	}

	// overwrite config settings with command line arguments
	if flagIsSet("serviceurl") {
		cfg.ServiceURL = *serviceURL
	}
	if flagIsSet("realm") {
		cfg.Realm = *realm
	}
	if flagIsSet("keepalive") {
		cfg.KeepAlive = *keepAlive
	}
	if flagIsSet("timeout") {
		cfg.Timeout = *timeout
	}
	if flagIsSet("retrytimer") {
		cfg.RetryTimer = *retryTimer
	}
	if flagIsSet("tndservers") {
		servers, ok := parseTNDServers(*tndServers)
		if !ok {
			log.WithField("tndservers", *tndServers).Fatal("Agent could not parse TND servers")
		}
		cfg.TND.HTTPSServers = servers
	}
	if flagIsSet("verbose") {
		cfg.Verbose = *verbose
	}
	if flagIsSet("minuserid") {
		cfg.MinUserID = *minUserID
	}
	if flagIsSet("startdelay") {
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
