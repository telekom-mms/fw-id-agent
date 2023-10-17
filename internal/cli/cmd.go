// Package cli contains the agent command line interface.
package cli

import (
	"flag"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/fw-id-agent/internal/agent"
	"github.com/telekom-mms/fw-id-agent/pkg/client"
	"github.com/telekom-mms/fw-id-agent/pkg/status"
)

var (
	// command is the command specified on the command line
	command = ""

	// verbose specifies verbose output
	verbose = false

	// json specifies whether output should be formatted as json
	json = false
)

// parseCommandLine parses the command line arguments
func parseCommandLine() {
	// status subcommand
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	statusCmd.BoolVar(&verbose, "verbose", verbose, "set verbose output")
	statusCmd.BoolVar(&json, "json", json, "set json output")

	// command line arguments
	ver := flag.Bool("version", false, "print version")

	// usage output
	flag.Usage = func() {
		cmd := os.Args[0]
		w := flag.CommandLine.Output()
		usage := func(f string, args ...any) {
			_, err := fmt.Fprintf(w, f, args...)
			if err != nil {
				log.WithError(err).Fatal("CLI could not print usage")
			}
		}
		usage("Usage:\n")
		usage("  %s [options] [command]\n", cmd)
		usage("\nOptions:\n")
		flag.PrintDefaults()
		usage("\nCommands:\n")
		usage("  status\n")
		usage("        show agent status\n")
		usage("  monitor\n")
		usage("        monitor agent status updates\n")
		usage("  relogin\n")
		usage("        relogin agent\n")
	}

	// parse command line arguments
	flag.Parse()

	// print version?
	if *ver {
		fmt.Println(agent.Version)
		os.Exit(0)
	}

	// parse subcommands
	command = flag.Arg(0)
	switch command {
	case "status":
		_ = statusCmd.Parse(os.Args[2:])
	case "monitor":
	case "relogin":
	default:
		flag.Usage()
		os.Exit(2)
	}
}

// printStatus prints status
func printStatus(s *status.Status, verbose bool) {
	fmt.Printf("Trusted Network:    %s\n", s.TrustedNetwork)
	fmt.Printf("Login State:        %s\n", s.LoginState)
	if verbose {
		// last keep-alive info
		lastKeepAlive := time.Unix(s.LastKeepAlive, 0)
		if lastKeepAlive.IsZero() {
			fmt.Printf("Last Keep-Alive:    0\n")
		} else {
			fmt.Printf("Last Keep-Alive:    %s\n", lastKeepAlive)
		}

		// kerberos info
		fmt.Printf("Kerberos TGT:\n")

		// kerberos tgt start time
		tgtStartTime := time.Unix(s.KerberosTGT.StartTime, 0)
		if tgtStartTime.IsZero() {
			fmt.Printf("- Start Time:       0\n")
		} else {
			fmt.Printf("- Start Time:       %s\n", tgtStartTime)
		}

		// kerberos tgt end time
		tgtEndTime := time.Unix(s.KerberosTGT.EndTime, 0)
		if tgtEndTime.IsZero() {
			fmt.Printf("- End Time:         0\n")
		} else {
			fmt.Printf("- End Time:         %s\n", tgtEndTime)
		}

		// agent config
		config, err := s.Config.JSON()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Config:             %s\n", config)
	}
}

// getStatus retrieves the agent status and prints it
func getStatus() {
	// create client
	c, err := client.NewClient()
	if err != nil {
		log.WithError(err).Fatal("could not create client")
	}
	defer func() { _ = c.Close() }()

	// query status
	s, err := c.Query()
	if err != nil {
		log.Fatal(err)
	}

	if json {
		// print status as json
		j, err := s.JSONIndent()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
		return
	}

	// print status
	printStatus(s, verbose)
}

// relogin sends a relogin request to the agent
func relogin() {
	// create client
	c, err := client.NewClient()
	if err != nil {
		log.WithError(err).Fatal("could not create client")
	}
	defer func() { _ = c.Close() }()

	// send request to agent
	if err := c.ReLogin(); err != nil {
		log.WithError(err).Error("re-login request failed")
	}
}

// monitor subscribes to status updates from the agent and displays them
func monitor() {
	// create client
	c, err := client.NewClient()
	if err != nil {
		log.WithError(err).Fatal("could not create client")
	}
	defer func() { _ = c.Close() }()

	// get status updates
	updates, err := c.Subscribe()
	if err != nil {
		log.WithError(err).Fatal("error subscribing to status updates")
	}
	for u := range updates {
		log.Println("Got status update:")
		printStatus(u, true)
	}
}

// Run is the main entry point
func Run() {
	parseCommandLine()

	switch command {
	case "status":
		getStatus()
	case "monitor":
		monitor()
	case "relogin":
		relogin()
	}
}
