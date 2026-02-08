package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"io"
	"strconv"
	"strings"
)

type pos []int

func (p *pos) newLine() token.Pos {
	l := len(*p)
	*p = append(*p, len(*p), len(*p)+1)

	return token.Pos(l + 1)
}

type constructor struct {
	pkg *types.Package
	pos
	types                       map[*types.Named][2]string
	statements                  []ast.Stmt
	needPtr, needSlice, needMap bool
}

func constructFile(w io.Writer, pkgName string, assigner, marshaler, unmarshaler, writer, reader string, opts []string, pkg *types.Package, typenames ...string) error {
	var typs []*types.Named

	for _, typename := range typenames {
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

		typs = append(typs, named)
	}

	c := constructor{
		pkg:   pkg,
		pos:   pos{0},
		types: make(map[*types.Named][2]string),
	}
	file := &ast.File{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{
					Slash: c.newLine(),
					Text:  "//go:generate go run vimagination.zapto.org/unsafe@latest " + encodeOpts(opts),
				},
			},
		},
		Name:    ast.NewIdent(pkgName),
		Package: c.newLine(),
		Decls:   c.buildDecls(assigner, marshaler, unmarshaler, writer, reader, typs),
	}
	fset := token.NewFileSet()
	wsfile := fset.AddFile("out.go", 1, len(c.pos))

	wsfile.SetLines(c.pos)

	return format.Node(w, fset, file)
}

func encodeOpts(opts []string) string {
	var buf []byte

	for n, opt := range opts {
		if n > 0 {
			buf = append(buf, ' ')
		}

		if strings.Contains(opt, " ") {
			buf = strconv.AppendQuote(buf, opt)
		} else {
			buf = append(buf, opt...)
		}
	}

	return string(buf)
}

func (c *constructor) buildDecls(assigner, marshaler, unmarshaler, writer, reader string, types []*types.Named) []ast.Decl {
	decls := []ast.Decl{
		c.imports(writer != "" || reader != ""),
	}

	for _, typ := range types {
		typeName := typ.Obj().Name()
		marshalName := marshalName(typ)
		unmarshalName := unmarshalName(typ)
		c.types[typ] = [2]string{marshalName, unmarshalName}

		if assigner != "" {
			decls = append(decls, c.assignBinary(typeName, assigner, marshalName))
		}

		if marshaler != "" {
			decls = append(decls, c.marshalBinary(typeName, marshaler, marshalName))
		}

		if writer != "" {
			decls = append(decls, c.writeTo(typeName, writer, marshalName))
		}

		if unmarshaler != "" {
			decls = append(decls, c.unmarshalBinary(typeName, unmarshaler, unmarshalName))
		}

		if reader != "" {
			decls = append(decls, c.readFrom(typeName, reader, unmarshalName))
		}
	}

	for _, typ := range types {
		if assigner != "" || marshaler != "" || writer != "" {
			decls = append(decls, c.marshalFunc(typ))
		}

		if unmarshaler != "" || reader != "" {
			decls = append(decls, c.unmarshalFunc(typ))
		}
	}

	if c.needPtr {
		decls = append(decls, newFunc())
	}

	if c.needSlice {
		decls = append(decls, makeSlice())
	}

	if c.needMap {
		decls = append(decls, makeMap()...)
	}

	return decls
}
