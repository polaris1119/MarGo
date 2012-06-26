package main

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"go/parser"
	"go/ast"
)

type ImportPathsArgs struct {
	Fn  string            `json:"fn"`
	Src string            `json:"src"`
	Env map[string]string `json:"env"`
}

type ImportPathsResult struct {
	Paths   []string     `json:"paths"`
	Imports []ImportDecl `json:"imports"`
}

func init() {
	act(Action{
		Path: "/import_paths",
		Doc:  "",
		Func: func(r Request) (data, error) {
			res := ImportPathsResult{
				Paths:   []string{},
				Imports: []ImportDecl{},
			}

			a := ImportPathsArgs{
				Env: map[string]string{},
			}

			if err := r.Decode(&a); err != nil {
				return res, err
			}

			_, af, err := parseAstFile(a.Fn, a.Src, parser.ImportsOnly)
			if err != nil {
				return res, err
			}

			for _, decl := range af.Decls {
				if gdecl, ok := decl.(*ast.GenDecl); ok && len(gdecl.Specs) > 0 {
					for _, spec := range gdecl.Specs {
						if ispec, ok := spec.(*ast.ImportSpec); ok {
							sd := ImportDecl{
								Path: unquote(ispec.Path.Value),
							}
							if ispec.Name != nil {
								sd.Name = ispec.Name.String()
							}
							res.Imports = append(res.Imports, sd)
						}
					}
				}
			}

			res.Paths, _ = importPaths(a.Env)
			return res, nil
		},
	})
}

func importPaths(environ map[string]string) ([]string, error) {
	imports := []string{}
	paths := map[string]bool{}

	env := []string{
		environ["GOPATH"],
		environ["GOROOT"],
		os.Getenv("GOPATH"),
		os.Getenv("GOROOT"),
		runtime.GOROOT(),
	}
	for _, ent := range env {
		for _, path := range filepath.SplitList(ent) {
			if path != "" {
				paths[path] = true
			}
		}
	}

	seen := map[string]bool{}
	pfx := strings.HasPrefix
	sfx := strings.HasSuffix
	osArch := runtime.GOOS + "_" + runtime.GOARCH
	for root, _ := range paths {
		root = filepath.Join(root, "pkg", osArch)
		walkF := func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				p, e := filepath.Rel(root, p)
				if e == nil && sfx(p, ".a") {
					p := p[:len(p)-2]
					if !pfx(p, ".") && !pfx(p, "_") && !sfx(p, "_test") {
						p = path.Clean(filepath.ToSlash(p))
						if !seen[p] {
							seen[p] = true
							imports = append(imports, p)
						}
					}
				}
			}
			return nil
		}
		filepath.Walk(root, walkF)
	}
	return imports, nil
}
