package main

import (
	"errors"
	"flag"
	"fmt"
	"go/types"
	"os"
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

	pkg, err := ParsePackage(module)
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

func processType(typ types.Type) Type {
	switch t := typ.Underlying().(type) {
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

func forStruct(t *types.Struct) Type {
	var s Struct

	for field := range t.NumFields() {
		s.Fields = append(s.Fields, Field{
			Name: t.Field(field).Name(),
			Type: processType(t.Field(field).Type()),
			Tag:  t.Tag(field),
		})
	}

	return s
}

func forArray(t *types.Array) Type {
	return Array{
		Length:  t.Len(),
		Element: processType(t.Elem()),
	}
}

func forSlice(t *types.Slice) Type {
	return Slice{
		Element: processType(t.Elem()),
	}
}

func forMap(t *types.Map) Type {
	return nil
}

var ErrNotFound = errors.New("typename not found")
