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

	Implements []bool
}

type processor struct {
	pkg     string
	methods []method
	named   map[string]*NamedType
}

type method struct {
	name    string
	args    []string
	returns []string
}

func (p *processor) processType(typ types.Type) Type {
	var nt NamedType

	if named, ok := typ.(*types.Named); ok {
		id := named.Obj().Id()

		if exist, ok := p.named[id]; ok {
			return exist
		}

		p.named[id] = &nt

		nt = NamedType{
			Package:    named.Obj().Pkg().Path(),
			Name:       named.Obj().Name(),
			Implements: make([]bool, len(p.methods)),
		}

		for method := range named.Methods() {
			name := method.Name()

		Loop:
			for i, req := range p.methods {
				if req.name != name {
					continue
				}

				sig := method.Signature()
				params := sig.Params()
				results := sig.Results()

				if params.Len() != len(req.args) || results.Len() != len(req.returns) {
					break
				}

				for n, arg := range req.args {
					if params.At(n).Type().String() != arg {
						break Loop
					}
				}

				for n, ret := range req.returns {
					if results.At(n).Type().String() != ret {
						break Loop
					}
				}

				nt.Implements[i] = true

				break
			}
		}
	}

	switch t := typ.Underlying().(type) {
	case *types.Struct:
		nt.Type = p.toStruct(t)
	case *types.Array:
		nt.Type = p.toArray(t)
	case *types.Slice:
		nt.Type = p.toSlice(t)
	case *types.Map:
		nt.Type = p.toMap(t)
	case *types.Pointer:
		nt.Type = p.toPointer(t)
	case *types.Basic:
		nt.Type = toBasic(t)
	}

	if nt.Name == "" {
		return nt.Type
	}

	return nt
}

func (p *processor) toStruct(t *types.Struct) Type {
	var s Struct

	for field := range t.NumFields() {
		s.Fields = append(s.Fields, Field{
			Name: t.Field(field).Name(),
			Type: p.processType(t.Field(field).Type()),
			Tag:  t.Tag(field),
		})
	}

	return s
}

func (p *processor) toArray(t *types.Array) Type {
	return Array{
		Length:  t.Len(),
		Element: p.processType(t.Elem()),
	}
}

func (p *processor) toSlice(t *types.Slice) Type {
	return Slice{
		Element: p.processType(t.Elem()),
	}
}

func (p *processor) toMap(t *types.Map) Type {
	return Map{
		Key:   p.processType(t.Key()),
		Value: p.processType(t.Elem()),
	}
}

func (p *processor) toPointer(t *types.Pointer) Type {
	return Pointer{
		Element: p.processType(t.Elem()),
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
