package options

import (
	"io"

	"github.com/pborman/getopt/v2"
)

// PrintUsage calls PrintUsage in the default option set.
func PrintUsage(w io.Writer) { getopt.PrintUsage(w) }

// Usage calls the usage function in the default option set.
func Usage() { getopt.Usage() }

// SetParameters sets the parameters string for printing the command line
// usage.  It defaults to "[parameters ...]"
func SetParameters(parameters string) {
	getopt.SetParameters(parameters)
}

// SetProgram sets the program name to program.  Normally it is determined
// from the zeroth command line argument (see os.Args).
func SetProgram(program string) {
	getopt.SetProgram(program)
}

// SetUsage sets the function used by Parse to display the commands usage
// on error.  It defaults to calling PrintUsage(os.Stderr).
func SetUsage(usage func()) {
	getopt.SetUsage(usage)
}

// SetDisplayWidth sets the width of the display when printing usage.  It defaults to 80.
func SetDisplayWidth(w int) {
	getopt.DisplayWidth = w
}

// SetHelpColumn sets the maximum column position that help strings start to
// display at. If the option usage is too long then the help string will be
// displayed on the next line.  It defaults to 20.
func SetHelpColumn(c int) {
	getopt.HelpColumn = c
}
