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
		return toStruct(t)
	case *types.Array:
		return toArray(t)
	case *types.Slice:
		return toSlice(t)
	case *types.Map:
		return toMap(t)
	case *types.Pointer:
		return toPointer(t)
	case *types.Basic:
		return toBasic(t)
	}

	return nil
}

func toStruct(t *types.Struct) Type {
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

func toArray(t *types.Array) Type {
	return Array{
		Length:  t.Len(),
		Element: processType(t.Elem()),
	}
}

func toSlice(t *types.Slice) Type {
	return Slice{
		Element: processType(t.Elem()),
	}
}

func toMap(t *types.Map) Type {
	return Map{
		Key:   processType(t.Key()),
		Value: processType(t.Elem()),
	}
}

func toPointer(t *types.Pointer) Type {
	return Pointer{
		Element: processType(t.Elem()),
	}
}

func toBasic(t *types.Basic) Type {
	switch t.Kind() {
	case types.Bool:
		return Bool{}
	case types.Int:
		return Int(0)
	case types.Int8:
		return Int(8)
	case types.Int16:
		return Int(16)
	case types.Int32:
		return Int(32)
	case types.Int64:
		return Int(64)
	case types.Uint:
		return Uint(0)
	case types.Uint8:
		return Uint(8)
	case types.Uint16:
		return Uint(16)
	case types.Uint32:
		return Uint(32)
	case types.Uint64, types.Uintptr:
		return Uint(64)
	case types.Float32:
		return Float(32)
	case types.Float64:
		return Float(64)
	case types.Complex64:
		return Complex(64)
	case types.Complex128:
		return Complex(128)
	case types.String:
		return String{}
	}

	return nil
}

var ErrNotFound = errors.New("typename not found")
