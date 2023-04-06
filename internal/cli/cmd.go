package cli

import (
	"github.com/T-Systems-MMS/fw-id-agent/internal/api"
	"github.com/T-Systems-MMS/fw-id-agent/internal/status"
	log "github.com/sirupsen/logrus"
)

// Run is the main entry point
func Run() {
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
