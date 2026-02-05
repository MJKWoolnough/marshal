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

type method struct {
	flag     string
	value    string
	disabled bool
}

func newMethodFlag(flagName, value string) *method {
	m := &method{
		flag:  flagName,
		value: value,
	}

	flag.StringVar(&m.value, flagName, value, "alternate name for the "+value+" method")
	flag.BoolVar(&m.disabled, "n"+flagName, false, "disable "+value+"method")

	return m
}

func run() error {
	var output string

	methods := []*method{
		newMethodFlag("w", "WriteTo"),
		newMethodFlag("r", "ReadFrom"),
		newMethodFlag("a", "AppendBinary"),
		newMethodFlag("m", "MarshalBinary"),
		newMethodFlag("u", "UnmarshalBinary"),
	}

	flag.StringVar(&output, "o", "", "output file")

	flag.Parse()

	if output == "" {
		return ErrNoOutput
	}

	for _, m := range methods {
		if m.disabled {
			m.value = ""
		}
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

		if named.TypeArgs().Len() != 0 {
			return fmt.Errorf("%w: %s", ErrGenericType, typename)
		}

		requested = append(requested, named)
	}

	args := append([]string{"-o", filepath.Base(output)}, flag.Args()...)

	fw := fileWriter{path: output}

	if err := constructFile(&fw, pkg.Name(), methods[2].value, methods[3].value, methods[4].value, methods[0].value, methods[1].value, args, requested...); err != nil {
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
	ErrNoOutput    = errors.New("no output file")
	ErrNotFound    = errors.New("typename not found")
	ErrNotAType    = errors.New("identifier is not a named type")
	ErrGenericType = errors.New("generic types are currently unsupported")
)
