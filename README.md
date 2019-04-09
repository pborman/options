# options ![build status](https://travis-ci.org/pborman/options.svg?branch=master)

Structured getopt processing for Go programs using the github.com/pborman/getopt/v2 package.

The options package makes adding getopt style command line options to Go programs as easy as declaring a structure:

```
package main

import (
	"fmt"
	"time"

	"github.com/pborman/options"
)

var opts = struct {
	Help    options.Help  `getopt:"--help           display help"`
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
	args := options.RegisterAndParse(&opts)

	if opts.Verbose {
		fmt.Printf("Command line parameters: %q\n", args)
	}
	fmt.Printf("Name: %s\n", opts.Name)
}
```

The options.Help type causes the command's usage to be displayed to standard error and the command to exit when the option is parsed from the command line.

The options package also supports reading options from file specified on the command line or an optional defaults file:

```
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/options"
)

var opts = struct {
	Flags   options.Flags `getopt:"--flags=PATH     read options from PATH"`
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
	options.Register(&opts)
	// read defaults from ~/.example.flags if the file exists.
	if err := opts.Flags.Set("?${HOME}/.example.flags", nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	args := options.Parse()

	if opts.Verbose {
		fmt.Printf("Command line parameters: %q\n", args)
	}
	fmt.Printf("Name: %s\n", opts.Name)
}
```

Using the following .example.flags file in your home directory that contains:

```
v = true
name = "github user"
```

The above program produces the following output:

```
$ go run x.go a parameter
Command line parameters: ["a" "parameter"]
Name: github user

$ go run x.go --help     
unknown option: --help
Usage: x [-v] [-c COUNT] [--flags PATH] [--lazy value] [-n NUMBER] [--name NAME] [--timeout value] [parameters ...]
 -c, --count=COUNT  number of widgets
     --flags=PATH   read options from PATH
     --lazy=value   unspecified
 -n NUMBER          set n to NUMBER
     --name=NAME    name of the widget [gopher]
     --timeout=value
                    duration of run
 -v                 be verbose
exit status 1
```
