package main

import (
	"log"
	"os"
	"strings"
	"time"
)

func init() {
	act(Action{
		Path: "/",
		Doc: `
expects data to be a string
returns {"actions": {"ACTION_PATH": "ACTION_DOC"}, "motd": "[the value passed in as data]"}
additionally, if data is "bye ni" MarGo will exit
`,
		Func: func(r Request) (data, error) {
			a := ""
			err := r.Decode(&a)
			if _, ok := err.(NoInputErr); ok {
				err = nil
			}
			if strings.TrimSpace(strings.ToLower(a)) == "bye ni" {
				go func() {
					time.Sleep(1 * time.Second)
					log.Println("exiting...")
					os.Exit(0)
				}()
			}
			m := map[string]string{}
			for path, ac := range actions {
				m[path] = ac.Doc
			}
			res := map[string]data{
				"actions": m,
				"motd":    a,
			}
			return res, err
		},
	})
}
