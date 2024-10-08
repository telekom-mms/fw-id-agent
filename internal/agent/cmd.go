package agent

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/fw-id-agent/pkg/config"
)

var (
	// Version is the agent version, to be set at compile time.
	Version = "unknown"
)

// command line argument names.
const (
	argConfig        = "config"
	argVersion       = "version"
	argServiceURL    = "serviceurl"
	argRealm         = "realm"
	argKeepAlive     = "keepalive"
	argLoginTimeout  = "logintimeout"
	argLogoutTimeout = "logouttimeout"
	argRetryTimer    = "retrytimer"
	argTNDServers    = "tndservers"
	argVerbose       = "verbose"
	argStartDelay    = "startdelay"
	argNotifications = "notifications"
)

// flagIsSet returns whether flag with name is set as command line argument.
func flagIsSet(flags *flag.FlagSet, name string) bool {
	isSet := false
	flags.Visit(func(f *flag.Flag) {
		if name == f.Name {
			isSet = true
		}
	})
	return isSet
}

// parseTNDServers parses the TND servers command line argument.
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

// getConfig gets the config from the config file and command line arguments.
func getConfig(args []string) (*config.Config, error) {
	// parse command line arguments
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	defaults := config.Default()
	cfgFile := flags.String(argConfig, "", "Set config `file`")
	ver := flags.Bool(argVersion, false, "print version")
	serviceURL := flags.String(argServiceURL, "", "Set service URL")
	realm := flags.String(argRealm, "", "Set kerberos realm")
	keepAlive := flags.Int(argKeepAlive, defaults.KeepAlive, "Set default client keep-alive in `minutes`")
	loginTimeout := flags.Int(argLoginTimeout, defaults.LoginTimeout, "Set client login request timeout in `seconds`")
	logoutTimeout := flags.Int(argLogoutTimeout, defaults.LogoutTimeout, "Set client logout request timeout in `seconds`")
	retryTimer := flags.Int(argRetryTimer, defaults.RetryTimer, "Set client login retry timer in case of errors in `seconds`")
	tndServers := flags.String(argTNDServers, "", "Set comma-separated `list` of TND server url:hash pairs")
	verbose := flags.Bool(argVerbose, defaults.Verbose, "Set verbose output")
	startDelay := flags.Int(argStartDelay, defaults.StartDelay, "Set agent start delay in `seconds`")
	notifications := flags.Bool(argNotifications, defaults.Notifications, "Set desktop notifications")
	if err := flags.Parse(args[1:]); err != nil {
		return nil, err
	}

	// print version?
	if *ver {
		fmt.Println(Version)
		return nil, flag.ErrHelp
	}

	// log version
	log.WithField("version", Version).Info("Starting Agent")

	// load config or try defaults
	cfg := config.Default()
	if flagIsSet(flags, argConfig) {
		c, err := config.Load(*cfgFile)
		if err != nil {
			return nil, fmt.Errorf("could not load config: %w", err)
		}
		cfg = c
	}

	// overwrite config settings with command line arguments
	if flagIsSet(flags, argServiceURL) {
		cfg.ServiceURL = *serviceURL
	}
	if flagIsSet(flags, argRealm) {
		cfg.Realm = *realm
	}
	if flagIsSet(flags, argKeepAlive) {
		cfg.KeepAlive = *keepAlive
	}
	if flagIsSet(flags, argLoginTimeout) {
		cfg.LoginTimeout = *loginTimeout
	}
	if flagIsSet(flags, argLogoutTimeout) {
		cfg.LogoutTimeout = *logoutTimeout
	}
	if flagIsSet(flags, argRetryTimer) {
		cfg.RetryTimer = *retryTimer
	}
	if flagIsSet(flags, argTNDServers) {
		servers, ok := parseTNDServers(*tndServers)
		if !ok {
			return nil, fmt.Errorf("could not parse TND servers %#v", *tndServers)
		}
		cfg.TND.HTTPSServers = servers
	}
	if flagIsSet(flags, argVerbose) {
		cfg.Verbose = *verbose
	}
	if flagIsSet(flags, argStartDelay) {
		cfg.StartDelay = *startDelay
	}
	if flagIsSet(flags, argNotifications) {
		cfg.Notifications = *notifications
	}

	// check if config is valid
	if !cfg.Valid() {
		return nil, fmt.Errorf("could not get valid config from file or command line arguments")
	}

	return cfg, nil
}

// setVerbose sets verbose mode based on the configuration.
func setVerbose(cfg *config.Config) {
	if cfg.Verbose {
		log.SetLevel(log.DebugLevel)
	}
}

func run(args []string) error {
	// get config
	cfg, err := getConfig(args)
	if err != nil {
		return err
	}

	// set verbose output
	setVerbose(cfg)

	log.WithField("config", cfg).Debug("Agent starting with valid config")

	// give the user's desktop environment some time to start after login,
	// so we do not miss notifications
	log.WithField("seconds", cfg.StartDelay).Debug("Agent sleeping before starting")
	time.Sleep(cfg.GetStartDelay())

	// start agent
	log.Debug("Agent starting")
	a := NewAgent(cfg)
	if err := a.Start(); err != nil {
		return fmt.Errorf("Agent could not start: %w", err)
	}
	defer a.Stop()

	// catch interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// wait for interrupt signal or agent error
	select {
	case <-c:
		log.Info("Agent got interrupt signal")
	case err = <-a.Errors():
	}

	return err
}

// Run is the main entry point.
func Run() {
	if err := run(os.Args); err != nil {
		if err != flag.ErrHelp {
			log.Fatal(err)
		}
		return
	}
}
