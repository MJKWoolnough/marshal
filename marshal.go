package main // import "vimagination.zapto.org/marshal"

import (
	"errors"
	"flag"
	"fmt"
	"go/types"
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

	typ := pkgs[0].Types.Scope().Lookup(typename)
	if typ == nil {
		return ErrNotFound
	}

	switch t := typ.Type().Underlying().(type) {
	case *types.Struct:
		return forStruct(t)
	case *types.Array:
		return forArray(t)
	case *types.Slice:
		return forSlice(t)
	case *types.Map:
		return forMap(t)
	}

	return nil
}

func forStruct(t *types.Struct) error {
	return nil
}

func forArray(t *types.Array) error {
	return nil
}

func forSlice(t *types.Slice) error {
	return nil
}

func forMap(t *types.Map) error {
	return nil
}

var ErrNotFound = errors.New("typename not found")
