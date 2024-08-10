package m

// NOTE: This file should be identical to twin/panicHandler.go

import (
	log "github.com/sirupsen/logrus"
)

func panicHandler(goroutineName string, recoverResult any) {
	if recoverResult == nil {
		return
	}

	log.WithFields(log.Fields{
		"recoverResult": recoverResult,
	}).Error("Goroutine panicked: " + goroutineName)
}
