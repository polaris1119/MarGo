package main

func init() {
	methods["hello"] = func(r Request) (data, error) {
		var a data
		err := r.Decode(&a)
		return a, err
	}
}
