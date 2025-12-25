package main

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
	Name string
	Type Type

	ImplementsAppendJSON      bool
	ImplementsMarshalJSON     bool
	ImplementsUnmarshalJSON   bool
	ImplementsAppendBinary    bool
	ImplementsMarshalBinary   bool
	ImplementsUnmarshalBinary bool
	ImplementsWriteTo         bool
	ImplementsReadFrom        bool
}
