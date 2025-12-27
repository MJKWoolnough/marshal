package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func run() error {
	var typename, module, output string

	flag.StringVar(&typename, "type", "", "typename to provide marshal/unmarshal functions for")
	flag.StringVar(&module, "module", "", "path to local module")
	flag.StringVar(&output, "o", "", "output file")

	flag.Parse()

	ignore, err := ignoreOutputFile(module, output)
	if err != nil {
		return err
	}

	pkg, err := ParsePackage(module, ignore...)
	if err != nil {
		return err
	}

	typ := pkg.Scope().Lookup(typename)
	if typ == nil {
		return ErrNotFound
	}

	processType(typ.Type())

	return nil
}

func ignoreOutputFile(module, output string) ([]string, error) {
	if output == "" || output == "-" {
		return nil, nil
	}

	var ignore []string

	o, err := filepath.Abs(output)
	if err != nil {
		return nil, err
	}

	m, err := filepath.Abs(module)
	if err != nil {
		return nil, err
	}

	if filepath.Dir(o) == filepath.Clean(m) {
		ignore = []string{filepath.Base(o)}
	}

	return ignore, nil
}

var ErrNotFound = errors.New("typename not found")
