package main

import (
	"bytes"
	"testing"
)

func TestRunRejectsUnexpectedArguments(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"unexpected"}, &output); err == nil {
		t.Fatal("run accepted an unexpected positional argument")
	}
}
