package main

import (
	"go/types"
)

type Type interface{}

type Struct struct {
	Fields []Field
}

type Field struct {
	Name string
	Type Type
	Tag  string
}

type Array struct {
	Length  int64
	Element Type
}

type Slice struct {
	Element Type
}

type Map struct {
	Key, Value Type
}

type Pointer struct {
	Element Type
}

type Int uint8

type Uint uint8

type Float uint8

type Complex uint8

type String struct{}

type Bool struct{}

type Interface struct{}

type NamedType struct {
	Package string
	Name    string
	Type    Type

	ImplementsAppendJSON      bool
	ImplementsMarshalJSON     bool
	ImplementsUnmarshalJSON   bool
	ImplementsAppendBinary    bool
	ImplementsMarshalBinary   bool
	ImplementsUnmarshalBinary bool
	ImplementsWriteTo         bool
	ImplementsReadFrom        bool
}

func processType(typ types.Type) Type {
	var obj Type

	switch t := typ.Underlying().(type) {
	case *types.Struct:
		obj = toStruct(t)
	case *types.Array:
		obj = toArray(t)
	case *types.Slice:
		obj = toSlice(t)
	case *types.Map:
		obj = toMap(t)
	case *types.Pointer:
		obj = toPointer(t)
	case *types.Basic:
		obj = toBasic(t)
	}

	if named, ok := typ.(*types.Named); ok {
		obj = NamedType{
			Package: named.Obj().Pkg().Path(),
			Name:    named.Obj().Name(),
			Type:    obj,
		}
	}

	return obj
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
