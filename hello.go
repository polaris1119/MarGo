package main

func init() {
	methods["hello"] = func(r *Request) (data, error) {
		return r, nil
	}
}
