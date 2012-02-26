package main

import (
	"os"
)

func init() {
	methods["env"] = func(r *Request) (data, error) {
		return os.Environ(), nil
	}
}
