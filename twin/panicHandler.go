package twin

// NOTE: This file should be identical to m/panicHandler.go

import (
	log "github.com/sirupsen/logrus"
)

func panicHandler(goroutineName string, recoverResult any, stackTrace []byte) {
	if recoverResult == nil {
		return
	}

	log.WithFields(log.Fields{
		"panic":      recoverResult,
		"stackTrace": string(stackTrace),
	}).Error("Goroutine panicked: " + goroutineName)
}
