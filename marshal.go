package main

import (
	"errors"
	"flag"
	"fmt"
	"go/types"
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

	if output == "" {
		return ErrNoOutput
	}

	pkg, err := gotypes.ParsePackage(filepath.Dir(output), output)
	if err != nil {
		return err
	}

	var requested []*types.Named

	for _, typename := range flag.Args() {
		typ := pkg.Scope().Lookup(typename)
		if typ == nil {
			return fmt.Errorf("%w: %s", ErrNotFound, typ)
		}

		named, ok := typ.Type().(*types.Named)
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotAType, typename)
		}

		requested = append(requested, named)
	}

	args := append([]string{"-o", filepath.Base(output)}, flag.Args()...)

	fw := fileWriter{path: output}

	if err := constructFile(&fw, pkg.Name(), "AppendBinary", "MarshalBinary", "UnmarshalBinary", "WriteTo", "ReadFrom", args, requested...); err != nil {
		return err
	}

	return fw.Close()
}

type fileWriter struct {
	path string
	*os.File
}

func (f *fileWriter) Write(p []byte) (int, error) {
	if f.File == nil {
		var err error

		f.File, err = os.Create(f.path)
		if err != nil {
			return 0, err
		}
	}

	return f.File.Write(p)
}

var (
	ErrNoOutput = errors.New("no output file")
	ErrNotFound = errors.New("typename not found")
	ErrNotAType = errors.New("identifier is not a named type")
)
