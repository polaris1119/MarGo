package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func init() {
	methods["import_paths"] = func(r Request) (data, error) {
		imports := []string{}
		paths := map[string]bool{}

		env := []string{
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
							imports = append(imports, p)
						}
					}
					if e != nil {
						println(e.Error())
					}
				}
				return nil
			}
			filepath.Walk(root, walkF)
		}
		return imports, nil
	}
}
