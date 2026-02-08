package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

func unmarshalName(typ *types.Named) string {
	return "_unmarshal_" + strings.ReplaceAll(strings.ReplaceAll(typ.Obj().Name(), "_", "__"), ".", "_")
}

func (c *constructor) unmarshalBinary(typeName, funcName, unmarshalName string) *ast.FuncDecl {
	comment := "// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface."

	if funcName != "UnmarshalBinary" {
		comment = "// " + funcName + " decodes the receiver from the binary form."
	}

	return &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{
					Slash: c.newLine(),
					Text:  comment,
				},
			},
		},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						ast.NewIdent("t"),
					},
					Type: &ast.UnaryExpr{
						Op: token.MUL,
						X:  ast.NewIdent(typeName),
					},
				},
			},
		},
		Name: &ast.Ident{
			Name: funcName,
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("b")},
						Type: &ast.ArrayType{
							Elt: ast.NewIdent("byte"),
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent("eb"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("byteio"),
								Sel: ast.NewIdent("MemLittleEndian"),
							},
							Args: []ast.Expr{
								ast.NewIdent("b"),
							},
						},
					},
				},
				&ast.ReturnStmt{
					Return: c.newLine(),
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: ast.NewIdent(unmarshalName),
							Args: []ast.Expr{
								ast.NewIdent("t"),
								&ast.UnaryExpr{
									Op: token.AND,
									X:  ast.NewIdent("eb"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *constructor) readFrom(typeName, funcName, unmarshalName string) *ast.FuncDecl {
	comment := "// ReadFrom implements the io.ReaderFrom interface."

	if funcName != "ReadFrom" {
		comment = "// " + funcName + " reads data from r until the type is fully decoded.\n//\n// The return value n is the number of bytes read. Any error encountered during the read is also returned."
	}

	return &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{
					Slash: c.newLine(),
					Text:  comment,
				},
			},
		},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						ast.NewIdent("t"),
					},
					Type: &ast.UnaryExpr{
						Op: token.MUL,
						X:  ast.NewIdent(typeName),
					},
				},
			},
		},
		Name: &ast.Ident{
			Name: funcName,
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{
							ast.NewIdent("r"),
						},
						Type: &ast.SelectorExpr{
							X:   ast.NewIdent("io"),
							Sel: ast.NewIdent("Reader"),
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent("int64"),
					},
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.TypeSwitchStmt{
					Assign: &ast.AssignStmt{
						Lhs: []ast.Expr{
							ast.NewIdent("r"),
						},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{
							&ast.TypeAssertExpr{
								X: ast.NewIdent("r"),
							},
						},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.CaseClause{
								List: []ast.Expr{
									&ast.UnaryExpr{
										Op: token.MUL,
										X: &ast.SelectorExpr{
											X:   ast.NewIdent("byteio"),
											Sel: ast.NewIdent("MemLittleEndian"),
										},
									},
								},
								Body: []ast.Stmt{
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											ast.NewIdent("l"),
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.CallExpr{
												Fun: ast.NewIdent("len"),
												Args: []ast.Expr{
													&ast.UnaryExpr{
														Op: token.MUL,
														X:  ast.NewIdent("r"),
													},
												},
											},
										},
									},
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											ast.NewIdent("err"),
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.CallExpr{
												Fun: ast.NewIdent(unmarshalName),
												Args: []ast.Expr{
													ast.NewIdent("t"),
													ast.NewIdent("r"),
												},
											},
										},
									},
									&ast.ReturnStmt{
										Return: c.newLine(),
										Results: []ast.Expr{
											&ast.CallExpr{
												Fun: ast.NewIdent("int64"),
												Args: []ast.Expr{
													&ast.BinaryExpr{
														X: &ast.CallExpr{
															Fun: ast.NewIdent("len"),
															Args: []ast.Expr{
																&ast.UnaryExpr{
																	Op: token.MUL,
																	X:  ast.NewIdent("r"),
																},
															},
														},
														Op: token.SUB,
														Y:  ast.NewIdent("l"),
													},
												},
											},
											ast.NewIdent("err"),
										},
									},
								},
							},
							&ast.CaseClause{
								List: []ast.Expr{
									&ast.UnaryExpr{
										Op: token.MUL,
										X: &ast.SelectorExpr{
											X:   ast.NewIdent("byteio"),
											Sel: ast.NewIdent("MemBigEndian"),
										},
									},
								},
								Body: []ast.Stmt{
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											ast.NewIdent("l"),
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.CallExpr{
												Fun: ast.NewIdent("len"),
												Args: []ast.Expr{
													&ast.UnaryExpr{
														Op: token.MUL,
														X:  ast.NewIdent("r"),
													},
												},
											},
										},
									},
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											ast.NewIdent("err"),
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.CallExpr{
												Fun: ast.NewIdent(unmarshalName),
												Args: []ast.Expr{
													ast.NewIdent("t"),
													ast.NewIdent("r"),
												},
											},
										},
									},
									&ast.ReturnStmt{
										Return: c.newLine(),
										Results: []ast.Expr{
											&ast.CallExpr{
												Fun: ast.NewIdent("int64"),
												Args: []ast.Expr{
													&ast.BinaryExpr{
														X: &ast.CallExpr{
															Fun: ast.NewIdent("len"),
															Args: []ast.Expr{
																&ast.UnaryExpr{
																	Op: token.MUL,
																	X:  ast.NewIdent("r"),
																},
															},
														},
														Op: token.SUB,
														Y:  ast.NewIdent("l"),
													},
												},
											},
											ast.NewIdent("err"),
										},
									},
								},
							},
							&ast.CaseClause{
								List: []ast.Expr{
									&ast.UnaryExpr{
										Op: token.MUL,
										X: &ast.SelectorExpr{
											X:   ast.NewIdent("byteio"),
											Sel: ast.NewIdent("StickyLittleEndianReader"),
										},
									},
								},
								Body: []ast.Stmt{
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											ast.NewIdent("l"),
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.SelectorExpr{
												X:   ast.NewIdent("r"),
												Sel: ast.NewIdent("Count"),
											},
										},
									},
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											&ast.SelectorExpr{
												X:   ast.NewIdent("r"),
												Sel: ast.NewIdent("Err"),
											},
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   ast.NewIdent("cmp"),
													Sel: ast.NewIdent("Or"),
												},
												Args: []ast.Expr{
													&ast.SelectorExpr{
														X:   ast.NewIdent("r"),
														Sel: ast.NewIdent("Err"),
													},
													&ast.CallExpr{
														Fun: ast.NewIdent(unmarshalName),
														Args: []ast.Expr{
															ast.NewIdent("t"),
															ast.NewIdent("r"),
														},
													},
												},
											},
										},
									},
									&ast.ReturnStmt{
										Return: c.newLine(),
										Results: []ast.Expr{
											&ast.BinaryExpr{
												X: &ast.SelectorExpr{
													X:   ast.NewIdent("r"),
													Sel: ast.NewIdent("Count"),
												},
												Op: token.SUB,
												Y:  ast.NewIdent("l"),
											},
											ast.NewIdent("err"),
										},
									},
								},
							},
							&ast.CaseClause{
								List: []ast.Expr{
									&ast.UnaryExpr{
										Op: token.MUL,
										X: &ast.SelectorExpr{
											X:   ast.NewIdent("byteio"),
											Sel: ast.NewIdent("StickyBigEndianReader"),
										},
									},
								},
								Body: []ast.Stmt{
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											ast.NewIdent("l"),
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.SelectorExpr{
												X:   ast.NewIdent("r"),
												Sel: ast.NewIdent("Count"),
											},
										},
									},
									&ast.AssignStmt{
										Lhs: []ast.Expr{
											&ast.SelectorExpr{
												X:   ast.NewIdent("r"),
												Sel: ast.NewIdent("Err"),
											},
										},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   ast.NewIdent("cmp"),
													Sel: ast.NewIdent("Or"),
												},
												Args: []ast.Expr{
													&ast.SelectorExpr{
														X:   ast.NewIdent("r"),
														Sel: ast.NewIdent("Err"),
													},
													&ast.CallExpr{
														Fun: ast.NewIdent(unmarshalName),
														Args: []ast.Expr{
															ast.NewIdent("t"),
															ast.NewIdent("r"),
														},
													},
												},
											},
										},
									},
									&ast.ReturnStmt{
										Return: c.newLine(),
										Results: []ast.Expr{
											&ast.BinaryExpr{
												X: &ast.SelectorExpr{
													X:   ast.NewIdent("r"),
													Sel: ast.NewIdent("Count"),
												},
												Op: token.SUB,
												Y:  ast.NewIdent("l"),
											},
											ast.NewIdent("err"),
										},
									},
								},
							},
						},
					},
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.Ident{
							NamePos: c.newLine(),
							Name:    "sr",
						},
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CompositeLit{
							Type: &ast.SelectorExpr{
								X:   ast.NewIdent("byteio"),
								Sel: ast.NewIdent("StickyLittleEndianReader"),
							},
							Elts: []ast.Expr{
								&ast.KeyValueExpr{
									Key:   ast.NewIdent("Reader"),
									Value: ast.NewIdent("r"),
								},
							},
						},
					},
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent("err"),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("cmp"),
								Sel: ast.NewIdent("Or"),
							},
							Args: []ast.Expr{
								&ast.CallExpr{
									Fun: ast.NewIdent(unmarshalName),
									Args: []ast.Expr{
										ast.NewIdent("t"),
										&ast.UnaryExpr{
											Op: token.AND,
											X:  ast.NewIdent("sr"),
										},
									},
								},
								&ast.SelectorExpr{
									X:   ast.NewIdent("sr"),
									Sel: ast.NewIdent("Err"),
								},
							},
						},
					},
				},
				&ast.ReturnStmt{
					Return: c.newLine(),
					Results: []ast.Expr{
						&ast.SelectorExpr{
							X:   ast.NewIdent("sr"),
							Sel: ast.NewIdent("Count"),
						},
						ast.NewIdent("err"),
					},
				},
			},
		},
	}
}

func (c *constructor) readType(name ast.Expr, typ types.Type) {
	switch t := typ.Underlying().(type) {
	case *types.Struct:
		c.readStruct(name, t)
	case *types.Array:
		c.readArray(name, t)
	case *types.Slice:
		c.readSlice(name, t)
	case *types.Map:
		c.readMap(name, t)
	case *types.Pointer:
		c.readPointer(name, t)
	case *types.Basic:
		c.readBasic(name, t)
	}
}

func (c *constructor) addCall(fun *ast.SelectorExpr, name ast.Expr) {
	c.addStatement(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  fun,
			Args: []ast.Expr{name},
		},
	})
}

func (c *constructor) addReader(method string, name ast.Expr) {
	c.addStatement(&ast.AssignStmt{
		Lhs: []ast.Expr{name},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("r"),
					Sel: ast.NewIdent(method),
				},
			},
		},
	})
}

func (c *constructor) readStruct(name ast.Expr, t *types.Struct) {
	for field := range t.Fields() {
		if !field.Exported() {
			continue
		}

		c.readType(&ast.SelectorExpr{
			X:   name,
			Sel: ast.NewIdent(field.Name()),
		}, field.Type())
	}
}

func (c *constructor) readArray(name ast.Expr, t *types.Array) {
	d := c.subConstructor()

	d.readType(&ast.IndexExpr{
		X:     name,
		Index: ast.NewIdent("n"),
	}, t.Elem())
	c.addStatement(&ast.RangeStmt{
		For: c.newLine(),
		Key: ast.NewIdent("n"),
		Tok: token.DEFINE,
		X:   name,
		Body: &ast.BlockStmt{
			List: d.statements,
		},
	})
}

func (c *constructor) readSlice(name ast.Expr, t *types.Slice) {
	c.makeSlice(name, t)
	c.readArray(name, types.NewArray(t.Elem(), 0))
}

func (c *constructor) makeSlice(name ast.Expr, t *types.Slice) {
	if typename := c.accessibleIdent(t.Elem()); typename != nil {
		c.addStatement(&ast.AssignStmt{
			Lhs: []ast.Expr{name},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: ast.NewIdent("make"),
					Args: []ast.Expr{
						&ast.ArrayType{
							Elt: typename,
						},
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("r"),
								Sel: ast.NewIdent("ReadUintX"),
							},
						},
					},
				},
			},
		})

		return
	}

	c.needSlice = true

	c.addStatement(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: ast.NewIdent("_make_slice"),
			Args: []ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X:  name,
				},
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("r"),
						Sel: ast.NewIdent("ReadUintX"),
					},
				},
			},
		},
	})
}

func makeSlice() *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("_make_slice"),
		Type: &ast.FuncType{
			TypeParams: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("T")},
						Type:  ast.NewIdent("any"),
					},
				},
			},
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("ptr")},
						Type: &ast.UnaryExpr{
							Op: token.MUL,
							X: &ast.ArrayType{
								Elt: ast.NewIdent("T"),
							},
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.MUL,
							X:  ast.NewIdent("ptr"),
						},
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: ast.NewIdent("make"),
							Args: []ast.Expr{
								&ast.ArrayType{
									Elt: ast.NewIdent("T"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *constructor) readMap(name ast.Expr, t *types.Map) {
	d := c.subConstructor()

	d.addStatement(c.makeMap(name, t))
	d.readType(ast.NewIdent("k"), t.Key())
	d.readType(ast.NewIdent("v"), t.Elem())
	d.addStatement(&ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.IndexExpr{
				X:     name,
				Index: ast.NewIdent("k"),
			},
		},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			ast.NewIdent("v"),
		},
	})

	c.addStatement(&ast.RangeStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("r"),
				Sel: ast.NewIdent("ReadUintX"),
			},
		},
		Body: &ast.BlockStmt{
			List: d.statements,
		},
	})
}

func (c *constructor) makeMap(name ast.Expr, t *types.Map) ast.Stmt {
	if keytypename, valuetypename := c.accessibleIdent(t.Key()), c.accessibleIdent(t.Elem()); keytypename != nil && valuetypename != nil {
		c.addStatement(&ast.AssignStmt{
			Lhs: []ast.Expr{name},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: ast.NewIdent("make"),
					Args: []ast.Expr{
						&ast.MapType{
							Key:   keytypename,
							Value: valuetypename,
						},
					},
				},
			},
		})

		return &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{
							ast.NewIdent("k"),
						},
						Type: keytypename,
					},
					&ast.ValueSpec{
						Names: []*ast.Ident{
							ast.NewIdent("v"),
						},
						Type: valuetypename,
					},
				},
			},
		}
	}

	c.needMap = true

	c.addStatement(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: ast.NewIdent("_make_map"),
			Args: []ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X:  name,
				},
			},
		},
	})

	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent("k"),
			ast.NewIdent("v"),
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: ast.NewIdent("_make_key_value"),
				Args: []ast.Expr{
					name,
				},
			},
		},
	}
}

func makeMap() []ast.Decl {
	return []ast.Decl{
		&ast.FuncDecl{
			Name: ast.NewIdent("_make_map"),
			Type: &ast.FuncType{
				TypeParams: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("K")},
							Type:  ast.NewIdent("comparable"),
						},
						{
							Names: []*ast.Ident{ast.NewIdent("V")},
							Type:  ast.NewIdent("any"),
						},
					},
				},
				Params: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("ptr")},
							Type: &ast.UnaryExpr{
								Op: token.MUL,
								X: &ast.MapType{
									Key:   ast.NewIdent("K"),
									Value: ast.NewIdent("V"),
								},
							},
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.UnaryExpr{
								Op: token.MUL,
								X:  ast.NewIdent("ptr"),
							},
						},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{
							&ast.CallExpr{
								Fun: ast.NewIdent("make"),
								Args: []ast.Expr{
									&ast.MapType{
										Key:   ast.NewIdent("K"),
										Value: ast.NewIdent("V"),
									},
								},
							},
						},
					},
				},
			},
		},
		&ast.FuncDecl{
			Name: ast.NewIdent("_map_key_value"),
			Type: &ast.FuncType{
				TypeParams: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("K")},
							Type:  ast.NewIdent("comparable"),
						},
						{
							Names: []*ast.Ident{ast.NewIdent("V")},
							Type:  ast.NewIdent("any"),
						},
					},
				},
				Params: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("_")},
							Type: &ast.MapType{
								Key:   ast.NewIdent("K"),
								Value: ast.NewIdent("V"),
							},
						},
					},
				},
				Results: &ast.FieldList{
					List: []*ast.Field{
						{
							Type: ast.NewIdent("K"),
						},
						{
							Type: ast.NewIdent("V"),
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{
										ast.NewIdent("k"),
									},
									Type: ast.NewIdent("K"),
								},
								&ast.ValueSpec{
									Names: []*ast.Ident{
										ast.NewIdent("v"),
									},
									Type: ast.NewIdent("V"),
								},
							},
						},
					},
					&ast.ReturnStmt{
						Results: []ast.Expr{
							ast.NewIdent("k"),
							ast.NewIdent("v"),
						},
					},
				},
			},
		},
	}
}

func (c *constructor) readPointer(name ast.Expr, t *types.Pointer) {
	d := c.subConstructor()

	c.new(name, t)
	d.readType(name, t.Elem())
	c.addStatement(&ast.IfStmt{
		Cond: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("r"),
				Sel: ast.NewIdent("ReadBool"),
			},
		},
		Body: &ast.BlockStmt{
			List: d.statements,
		},
	})
}

func (c *constructor) new(name ast.Expr, t *types.Pointer) {
	if typename := c.accessibleIdent(t.Elem()); typename != nil {
		c.addStatement(&ast.AssignStmt{
			Lhs: []ast.Expr{name},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.CallExpr{
				Fun:  ast.NewIdent("new"),
				Args: []ast.Expr{typename},
			}},
		})

		return
	}

	c.needPtr = true

	c.addStatement(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: ast.NewIdent("_new"),
			Args: []ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X:  name,
				},
			},
		},
	})
}

func newFunc() *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("_new"),
		Type: &ast.FuncType{
			TypeParams: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("T")},
						Type:  ast.NewIdent("any"),
					},
				},
			},
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("ptr")},
						Type: &ast.UnaryExpr{
							Op: token.MUL,
							X: &ast.UnaryExpr{
								Op: token.MUL,
								X:  ast.NewIdent("T"),
							},
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.MUL,
							X:  ast.NewIdent("ptr"),
						},
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: ast.NewIdent("new"),
							Args: []ast.Expr{
								ast.NewIdent("T"),
							},
						},
					},
				},
			},
		},
	}
}

func (c *constructor) accessibleIdent(t types.Type) ast.Expr {
	if named, ok := t.(*types.Named); ok && (named.Obj().Exported() || named.Obj().Pkg() == c.pkg || named.Obj().Pkg() == nil) {
		if named.Obj().Pkg() == c.pkg || named.Obj().Pkg() == nil {
			return ast.NewIdent(named.Obj().Name())
		}

		return &ast.SelectorExpr{
			X:   ast.NewIdent(named.Obj().Pkg().Name()),
			Sel: ast.NewIdent(named.Obj().Name()),
		}
	} else if basic, ok := t.Underlying().(*types.Basic); ok {
		return ast.NewIdent(basic.Name())
	}

	return nil
}

func (c *constructor) readBasic(name ast.Expr, t *types.Basic) {
	switch t.Kind() {
	case types.Bool:
		c.addReader("ReadBool", name)
	case types.Int:
		c.addReader("ReadInt64", name)
	case types.Int8:
		c.addReader("ReadInt8", name)
	case types.Int16:
		c.addReader("ReadInt16", name)
	case types.Int32:
		c.addReader("ReadInt32", name)
	case types.Int64:
		c.addReader("ReadInt64", name)
	case types.Uint:
		c.addReader("ReadUint64", name)
	case types.Uint8:
		c.addReader("ReadUint8", name)
	case types.Uint16:
		c.addReader("ReadUint16", name)
	case types.Uint32:
		c.addReader("ReadUint32", name)
	case types.Uint64, types.Uintptr:
		c.addReader("ReadUint64", name)
	case types.Float32:
		c.addReader("ReadFloat32", name)
	case types.Float64:
		c.addReader("ReadFloat64", name)
	case types.Complex64:
		c.addStatement(&ast.AssignStmt{
			Lhs: []ast.Expr{name},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: ast.NewIdent("complex"),
					Args: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("r"),
								Sel: ast.NewIdent("ReadFloat32"),
							},
						},
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("r"),
								Sel: ast.NewIdent("ReadFloat32"),
							},
						},
					},
				},
			},
		})
	case types.Complex128:
		c.addStatement(&ast.AssignStmt{
			Lhs: []ast.Expr{name},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: ast.NewIdent("complex"),
					Args: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("r"),
								Sel: ast.NewIdent("ReadFloat64"),
							},
						},
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("r"),
								Sel: ast.NewIdent("ReadFloat64"),
							},
						},
					},
				},
			},
		})
	case types.String:
		c.addReader("ReadStringX", name)
	}
}

func (c *constructor) unmarshalFunc(typ *types.Named) *ast.FuncDecl {
	typeName := typ.Obj().Name()
	unmarshalName := unmarshalName(typ)
	c.statements = nil

	c.readType(ast.NewIdent("t"), typ)

	return &ast.FuncDecl{
		Name: &ast.Ident{
			Name: unmarshalName,
		},
		Type: &ast.FuncType{
			Func: c.newLine(),
			TypeParams: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{
							ast.NewIdent("R"),
						},
						Type: &ast.SelectorExpr{
							X:   ast.NewIdent("byteio"),
							Sel: ast.NewIdent("StickyReader"),
						},
					},
				},
			},
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{
							ast.NewIdent("t"),
						},
						Type: &ast.UnaryExpr{
							Op: token.MUL,
							X:  ast.NewIdent(typeName),
						},
					},
					{
						Names: []*ast.Ident{
							ast.NewIdent("r"),
						},
						Type: ast.NewIdent("R"),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: ast.NewIdent("error"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: append(c.statements, &ast.ReturnStmt{
				Results: []ast.Expr{
					ast.NewIdent("nil"),
				},
			}),
		},
	}
}
