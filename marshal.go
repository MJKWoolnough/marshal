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
	var (
		output          string
		writeTo         = "WriteTo"
		readFrom        = "ReadFrom"
		appendBinary    = "AppendBinary"
		marshalBinary   = "MarshalBinary"
		unmarshalBinary = "UnmarshalBinary"
	)

	flag.StringVar(&writeTo, "w", writeTo, "alternate name for the WriteTo method")
	flag.StringVar(&readFrom, "r", readFrom, "alternate name for the ReadFrom method")
	flag.StringVar(&appendBinary, "a", appendBinary, "alternate name for the AppendBinary method")
	flag.StringVar(&marshalBinary, "m", marshalBinary, "alternate name for the MarshalBinary method")
	flag.StringVar(&unmarshalBinary, "u", unmarshalBinary, "alternate name for the UnmarshalBinary method")
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

	if err := constructFile(&fw, pkg.Name(), appendBinary, marshalBinary, unmarshalBinary, writeTo, readFrom, args, requested...); err != nil {
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
