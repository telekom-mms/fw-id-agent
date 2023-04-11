package cli

import (
	"flag"
	"fmt"

	"github.com/T-Systems-MMS/fw-id-agent/internal/api"
	"github.com/T-Systems-MMS/fw-id-agent/internal/status"
	log "github.com/sirupsen/logrus"
)

var (
	// command is the command specified on the command line
	command = ""

	// json specifies whether output should be formatted as json
	json = false
)

// parseCommandLine parses the command line arguments
func parseCommandLine() {
	flag.BoolVar(&json, "json", json, "set json output")
	flag.Parse()
	command = flag.Arg(0)
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
	fmt.Printf("Config:          %#v\n", *status.Config)
}

// Run is the main entry point
func Run() {
	parseCommandLine()

	switch command {
	case "status":
		getStatus()
	}
}
