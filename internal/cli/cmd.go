package cli

import (
	"flag"

	"github.com/T-Systems-MMS/fw-id-agent/internal/api"
	"github.com/T-Systems-MMS/fw-id-agent/internal/status"
	log "github.com/sirupsen/logrus"
)

var (
	// command is the command specified on the command line
	command = ""
)

// parseCommandLine parses the command line arguments
func parseCommandLine() {
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

	log.Printf("Trusted Network: %t", status.TrustedNetwork)
	log.Printf("Logged In: %t", status.LoggedIn)
	log.Printf("Config: %#v", *status.Config)
}

// Run is the main entry point
func Run() {
	parseCommandLine()

	switch command {
	case "status":
		getStatus()
	}
}
