package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type Doc struct {
	Src  string `json:"src"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Fn   string `json:"fn"`
	Row  int    `json:"row"`
	Col  int    `json:"col"`
}

type DocArgs struct {
	Fn        string            `json:"fn"`
	Src       string            `json:"src"`
	Expr      string            `json:"expr"`
	Env       map[string]string `json:"env"`
	Offset    int               `json:"offset"`
	TabIndent bool              `json:"tab_indent"`
	TabWidth  int               `json:"tab_width"`
}

func init() {
	act(Action{
		Path: "/doc",
		Doc:  "",
		Func: func(r Request) (data, error) {
			res := []*Doc{}

			a := DocArgs{Env: map[string]string{}}

			if err := r.Decode(&a); err != nil {
				return res, err
			}

			fset, af, err := parseAstFile(a.Fn, a.Src, parser.ParseComments)
			if err != nil {
				return res, err
			}

			sel, id := identAt(fset, af, a.Offset)

			pkg, pkgs, err := parsePkg(fset, filepath.Dir(a.Fn), parser.ParseComments)
			if pkg == nil {
				return nil, err
			}

			obj, _, objPkgs := findUnderlyingObj(fset, af, pkg, pkgs, rootDirs(a.Env), sel, id)
			if obj != nil {
				res = append(res, objDoc(fset, a.TabIndent, a.TabWidth, obj))
				if objPkgs != nil {
					xName := "Example" + obj.Name
					xPrefix := xName + "_"
					for _, objPkg := range objPkgs {
						xPkg, _ := ast.NewPackage(fset, objPkg.Files, nil, nil)
						if xPkg == nil || xPkg.Scope == nil {
							continue
						}

						for _, xObj := range xPkg.Scope.Objects {
							if xObj.Name == xName || strings.HasPrefix(xObj.Name, xPrefix) {
								res = append(res, objDoc(fset, a.TabIndent, a.TabWidth, xObj))
							}
						}
					}
				}
			}
			return res, nil
		},
	})
}

func objDoc(fset *token.FileSet, tabIndent bool, tabWidth int, obj *ast.Object) *Doc {
	objSrc, _ := printSrc(fset, obj.Decl, tabIndent, tabWidth)
	tp := fset.Position(obj.Pos())
	return &Doc{
		Src:  objSrc,
		Name: obj.Name,
		Kind: obj.Kind.String(),
		Fn:   tp.Filename,
		Row:  tp.Line - 1,
		Col:  tp.Column - 1,
	}
}

func isBetween(n, start, end int) bool {
	return (n >= start && n <= end)
}

func identAt(fset *token.FileSet, af *ast.File, offset int) (sel *ast.SelectorExpr, id *ast.Ident) {
	ast.Inspect(af, func(n ast.Node) bool {
		if n != nil {
			start := fset.Position(n.Pos())
			end := fset.Position(n.End())
			if isBetween(offset, start.Offset, end.Offset) {
				switch v := n.(type) {
				case *ast.SelectorExpr:
					sel = v
				case *ast.Ident:
					id = v
				}
			}
		}
		return true
	})
	return
}

func findUnderlyingObj(fset *token.FileSet, af *ast.File, pkg *ast.Package, pkgs map[string]*ast.Package, srcRootDirs []string, sel *ast.SelectorExpr, id *ast.Ident) (*ast.Object, *ast.Package, map[string]*ast.Package) {
	if id != nil && id.Obj != nil {
		return id.Obj, pkg, pkgs
	}

	if id == nil {
		// can this ever happen?
		return nil, pkg, pkgs
	}

	if sel == nil {
		if obj := pkg.Scope.Lookup(id.Name); obj != nil {
			return obj, pkg, pkgs
		}
		fn := filepath.Join(runtime.GOROOT(), "src", "pkg", "builtin")
		if pkgBuiltin, _, err := parsePkg(fset, fn, parser.ParseComments); err == nil {
			if obj := pkgBuiltin.Scope.Lookup(id.Name); obj != nil {
				return obj, pkgBuiltin, pkgs
			}
		}
	}

	if sel == nil {
		return nil, pkg, pkgs
	}

	switch x := sel.X.(type) {
	case *ast.Ident:
		if x.Obj != nil {
			// todo: resolve type
		} else {
			if v := pkg.Scope.Lookup(id.Name); v != nil {
				// todo: found a type?
			} else {
				// it's most likely a package
				// todo: handle .dot imports
				for _, ispec := range af.Imports {
					importPath := unquote(ispec.Path.Value)
					pkgAlias := ""
					if ispec.Name == nil {
						pkgAlias = path.Base(importPath)
					} else {
						pkgAlias = ispec.Name.Name
					}
					if pkgAlias == x.Name {
						if id == x {
							// where do we go as the first place of a package?
							return nil, pkg, pkgs
						}

						var p *ast.Package
						if p, pkgs, _ = findPkg(fset, importPath, srcRootDirs, parser.ParseComments); p != nil {
							obj := p.Scope.Lookup(id.Name)
							return obj, pkg, pkgs
						}
					}
				}
			}
		}
	}
	return nil, pkg, pkgs
}
