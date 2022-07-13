package notify

import (
	"github.com/gen2brain/beeep"
	log "github.com/sirupsen/logrus"
)

// Notify sends a notification to the user
func Notify(title, message string) {
	err := beeep.Notify(title, message, "")
	if err != nil {
		log.WithError(err).Error("Agent notify error")
	}
}
