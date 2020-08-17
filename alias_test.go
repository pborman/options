package options

import (
	"bytes"
	"io"
	"testing"

	"github.com/pborman/getopt/v2"
)

func TestSetVar(t *testing.T) {
	dw, hc, cl := getopt.DisplayWidth, getopt.HelpColumn, getopt.CommandLine
	defer func() {
		getopt.DisplayWidth, getopt.HelpColumn, getopt.CommandLine = dw, hc, cl
	}()
	SetDisplayWidth(42)
	SetHelpColumn(17)
	if getopt.DisplayWidth != 42 {
		t.Errorf("Setting DisplayWidth got %d, want 42", getopt.DisplayWidth)
	}
	if getopt.HelpColumn != 17 {
		t.Errorf("Setting HelpColumn got %d, want 17", getopt.HelpColumn)
	}
}

func TestUsage(t *testing.T) {
	// We screw up the usage function here so we can't test it further.
	var w io.Writer
	var buf bytes.Buffer
	SetUsage(func() { w = &buf })
	Usage()
	if w != &buf {
		t.Errorf("Usage did not call the function set by SetUsage")
	}

	cl := getopt.CommandLine
	defer func() { getopt.CommandLine = cl }()
	SetParameters("PARAMS")
	SetProgram("TEST")
	PrintUsage(&buf)
	got, want := buf.String(), "Usage: TEST PARAMS\n"
	if got != want {
		t.Errorf("Got usage %q, want %q", got, want)
	}
}
