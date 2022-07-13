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

const (
	// minUserID is the minimum allowed user ID
	// TODO: move this to config?
	minUserID = 1000

	// startDelay is the time the agent sleeps before starting in seconds
	// TODO: move this to config?
	startDelay = 20
)

var (
	// version is the agent version, to be set at compile time
	version = "unknown"
)

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

	// set verbose output
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	// load config
	cfg, err := config.Load(*cfgFile)
	if err != nil {
		log.WithError(err).Fatal("Agent could not load config")
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
	if uid < minUserID {
		log.Fatal("Agent started with user id lower than minimum allowed user id")
	}

	// give the user's desktop environment some time to start after login,
	// so we do not miss notifications
	log.WithField("seconds", startDelay).Debug("Agent sleeping before starting")
	time.Sleep(startDelay * time.Second)

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
