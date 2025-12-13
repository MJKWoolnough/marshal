package marshal // import "vimagination.zapto.org/marshal"

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func run() error {
	var typename string

	flag.StringVar(&typename, "type", "", "typename to provide marshal/unmarshall functions for")

	return processType(typename, flag.Args())
}

func processType(typename string, source []string) error {
	return nil
}
