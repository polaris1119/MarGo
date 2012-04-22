package main

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

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
						imports = append(imports, p)
					}
				}
			}
			return nil
		}
		filepath.Walk(root, walkF)
	}
	return imports, nil
}
