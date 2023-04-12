package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/T-Systems-MMS/fw-id-agent/internal/api"
	"github.com/T-Systems-MMS/fw-id-agent/internal/status"
	log "github.com/sirupsen/logrus"
)

var (
	// command is the command specified on the command line
	command = ""

	// verbose specifies verbose output
	verbose = false

	// json specifies whether output should be formatted as json
	json = false

	// version is the CLI version, to be set at compile time
	version = "unknown"
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
		usage := func(f string, args ...interface{}) {
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
		usage("  relogin\n")
		usage("        relogin agent\n")
	}

	// parse command line arguments
	flag.Parse()

	// print version?
	if *ver {
		fmt.Println(version)
		os.Exit(0)
	}

	// parse subcommands
	command = flag.Arg(0)
	switch command {
	case "status":
		statusCmd.Parse(os.Args[2:])
	case "relogin":
	default:
		flag.Usage()
		os.Exit(2)
	}
}

// getStatus retrieves the agent status and prints it
func getStatus() {
	client := api.NewClient(api.GetUserSocketFile())
	b := client.Query()
	if b == nil {
		return
	}
	status, err := status.NewFromJSON(b)
	if err != nil {
		log.Fatal(err)
	}

	if json {
		j, err := status.JSONIndent()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
		return
	}

	fmt.Printf("Trusted Network: %t\n", status.TrustedNetwork)
	fmt.Printf("Logged In:       %t\n", status.LoggedIn)
	if verbose {
		fmt.Printf("Config:          %#v\n", *status.Config)
	}
}

// relogin sends a relogin request to the agent
func relogin() {
	// send request to agent
	client := api.NewClient(api.GetUserSocketFile())
	msg := api.NewMessage(api.TypeRelogin, nil)
	reply := client.Request(msg)

	// handle response
	switch reply.Type {
	case api.TypeOK:
	case api.TypeError:
		log.WithField("error", string(reply.Value)).Error("Agent sent error reply")
	}
}

// Run is the main entry point
func Run() {
	parseCommandLine()

	switch command {
	case "status":
		getStatus()
	case "relogin":
		relogin()
	}
}
