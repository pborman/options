// Example usage of the options package.
package main

import (
	"time"

	"github.com/pborman/options"
)

var opts = struct {
	Help    options.Help  `getopt:"--help           display help"`
	Name    string        `getopt:"--name=NAME      name of the widget"`
	Count   int           `getopt:"--count -c=COUNT number of widgets"`
	Verbose bool          `getopt:"-v               be verbose"`
	Timeout time.Duration `getopt:"--timeout        duration of run"`
}{
	Name: "gopher",
}

func main() {
	options.RegisterAndParse(&opts)
}
