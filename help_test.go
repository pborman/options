package options

import (
	"os"
	"testing"

	"github.com/pborman/getopt/v2"
)

func TestHelpType(t *testing.T) {
	cl, args := getopt.CommandLine, os.Args
	defer func() { getopt.CommandLine, os.Args = cl, args }()
	var opts = &struct {
		H Help `getopt:"-? help"`
	}{H: true}
	os.Args = []string{"test", "-?"}
	RegisterAndParse(opts)
	v := opts.H.String()
	if v != "true" {
		t.Errorf("Got %v want true", v)
	}
}
