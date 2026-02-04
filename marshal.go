package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"vimagination.zapto.org/gotypes"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func run() error {
	var typename, output string

	flag.StringVar(&typename, "t", "", "typename to provide marshal/unmarshal functions for")
	flag.StringVar(&output, "o", "", "output file")

	flag.Parse()

	pkg, err := gotypes.ParsePackage(filepath.Dir(output), output)
	if err != nil {
		return err
	}

	typ := pkg.Scope().Lookup(typename)
	if typ == nil {
		return ErrNotFound
	}

	var p processor
	p.methods = []method{
		{
			name:    "WriteTo",
			args:    []string{"io.Writer"},
			returns: []string{"int64", "error"},
		},
	}
	p.named = map[string]*NamedType{}

	return nil
}

var ErrNotFound = errors.New("typename not found")
