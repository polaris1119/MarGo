package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"path/filepath"
)

type DeclarationsArgs struct {
	Fn  string `json:"filename"`
	Src string `json:"src"`
}

func init() {
	act(Action{
		Path: "/declarations",
		Doc:  "",
		Func: func(r Request) (data, error) {
			a := DeclarationsArgs{}
			decls := []map[string]interface{}{}
			if err := r.Decode(&a); err != nil {
				return decls, err
			}

			fset, _, err := parseAstFile(a.Fn, a.Src, 0)
			if err == nil {
				pkg, _, err := parsePkg(fset, filepath.Dir(a.Fn), 0)
				if pkg == nil {
					return decls, err
				}

				for _, af := range pkg.Files {
					for _, d := range af.Decls {
						if p := fset.Position(d.Pos()); p.IsValid() {
							switch n := d.(type) {
							case *ast.FuncDecl:
								if n.Name.Name != "_" {
									decl := map[string]interface{}{
										"name":     n.Name.Name,
										"kind":     "func",
										"doc":      n.Doc.Text(),
										"filename": p.Filename,
										"line":     p.Line,
										"column":   p.Column,
									}
									if n.Recv != nil {
										recvFields := n.Recv.List
										if len(recvFields) == 0 {
											break
										}
										typ := recvFields[0].Type
										buf := bytes.NewBuffer([]byte("("))
										if printer.Fprint(buf, fset, typ) != nil {
											break
										}
										fmt.Fprintf(buf, ").%s", n.Name.Name)
										decl["name"] = buf.String()
									}
									decls = append(decls, decl)
								}
							case *ast.GenDecl:
								for _, spec := range n.Specs {
									switch gn := spec.(type) {
									case *ast.TypeSpec:
										if vp := fset.Position(gn.Pos()); gn.Name.Name != "_" && vp.IsValid() {
											decls = append(decls, map[string]interface{}{
												"name":     gn.Name.Name,
												"kind":     "type",
												"doc":      gn.Doc.Text(),
												"filename": vp.Filename,
												"line":     vp.Line,
												"column":   vp.Column,
											})
										}
									case *ast.ValueSpec:
										for _, v := range gn.Names {
											if vp := fset.Position(v.Pos()); v.Name != "_" && vp.IsValid() {
												kind := ""
												switch v.Obj.Kind {
												case ast.Typ:
													kind = "type"
												case ast.Fun:
													kind = "func"
												case ast.Con:
													kind = "const"
												case ast.Var:
													kind = "var"
												default:
													continue
												}
												decls = append(decls, map[string]interface{}{
													"name":     v.Name,
													"kind":     kind,
													"doc":      "",
													"filename": vp.Filename,
													"line":     vp.Line,
													"column":   vp.Column,
												})
											}
										}
									}
								}
							}
						}
					}
				}
			}
			return decls, err
		},
	})
}
