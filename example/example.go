package main

import (
	"fmt"
	"time"

	"github.com/pborman/options"
	_ "github.com/pborman/options/json"
)

var theOptions = struct {
	Flags   options.Flags `getopt:"--flags=PATH     read default flags from PATH"`
	JFlags   options.Flags `getopt:"--json=PATH     read default flags from json blob at PATH" encoding:"json"`
	Name    string        `getopt:"--name=NAME      name of the widget"`
	Count   int           `getopt:"--count -c=COUNT number of widgets"`
	Verbose bool          `getopt:"-v               be verbose"`
	N       int           `getopt:"-n=NUMBER        set n to NUMBER"`
	Timeout time.Duration `getopt:"--timeout        duration of run"`
	Lazy    string
}{
	Name: "gopher",
}

func main() {
	theOptions.Flags.Set("?${HOME}/.example.flags", nil)
	theOptions.JFlags.Set("?${HOME}/.example.json", nil)

	args := options.RegisterAndParse(&theOptions)

	fmt.Printf("Arguments: %q\n", args)

	fmt.Printf("Flags: %v\n", theOptions.Flags.String())
	fmt.Printf("JFlags: %v\n", theOptions.JFlags.String())
	fmt.Printf("Name: %v\n", theOptions.Name)
	fmt.Printf("Count: %v\n", theOptions.Count)
	fmt.Printf("Verbose: %v\n", theOptions.Verbose)
	fmt.Printf("N: %v\n", theOptions.N)
	fmt.Printf("Timeout: %v\n", theOptions.Timeout)
	fmt.Printf("Lazy: %v\n", theOptions.Lazy)
}
