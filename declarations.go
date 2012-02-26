package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

func init() {
	methods["declarations"] = func(r *Request) (data, error) {
		decls := []map[string]interface{}{}
		fset := token.NewFileSet()
		var err error
		fn := r.Args.Filename
		var src interface{} = r.Args.Src
		if src.(string) == "" {
			src, err = os.Open(fn)
		}
		if err == nil {
			if fn == "" {
				fn = "<stdin>"
			}
			var af *ast.File
			af, err = parser.ParseFile(fset, fn, src, 0)
			if af != nil {
				for _, d := range af.Decls {
					if p := fset.Position(d.Pos()); p.IsValid() {
						switch n := d.(type) {
						case *ast.FuncDecl:
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
						case *ast.GenDecl:
							for _, spec := range n.Specs {
								switch gn := spec.(type) {
								case *ast.TypeSpec:
									if vp := fset.Position(gn.Pos()); vp.IsValid() {
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
										if vp := fset.Position(v.Pos()); vp.IsValid() {
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
	}
}
