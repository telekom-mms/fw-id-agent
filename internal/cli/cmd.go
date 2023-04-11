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

	// flags
	ver := flag.Bool("version", false, "print version")
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

// Run is the main entry point
func Run() {
	parseCommandLine()

	switch command {
	case "status":
		getStatus()
	}
}
