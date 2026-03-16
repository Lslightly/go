package main

var sink interface{} = nil

func main() {
	pa := new(int)
	foo(&pa)
}

//go:noinline
func foo(pp **int) {
	sink = *pp
}
