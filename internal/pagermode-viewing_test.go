package internal

import (
	"os"
	"testing"
)

func TestErrUnlessExecutable_yes(t *testing.T) {
	// Find our own executable
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	// Check that it's executable
	err = errUnlessExecutable(executable)
	if err != nil {
		t.Fatal(err)
	}
}

func TestErrUnlessExecutable_no(t *testing.T) {
	textFile := "pagermode-viewing_test.go"
	if _, err := os.Stat(textFile); os.IsNotExist(err) {
		t.Fatal("Test setup failed, text file not found: " + textFile)
	}

	err := errUnlessExecutable(textFile)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}
