package agent

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
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

// Run is the main entry point
func Run() {
	// parse command line arguments
	cfgFile := flag.String("config", "config.json", "Set config `file`")
	verbose := flag.Bool("verbose", false, "Set verbose output")
	ver := flag.Bool("version", false, "print version")
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
	if flagIsSet("verbose") {
		cfg.Verbose = *verbose
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
	time.Sleep(time.Duration(cfg.StartDelay) * time.Second)

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
