package main // import "vimagination.zapto.org/marshal"

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func run() error {
	var typename, module string

	flag.StringVar(&typename, "type", "", "typename to provide marshal/unmarshal functions for")
	flag.StringVar(&module, "module", "", "path to local module")

	flag.Parse()

	return processType(module, typename)
}

func processType(module, typename string) error {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadTypes}, module)
	if err != nil {
		return err
	}

	fmt.Println(pkgs[0].Types.Scope().Lookup(typename))

	return nil
}
