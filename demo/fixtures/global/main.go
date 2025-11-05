package main

var pi *int
var sinki interface{}

func main() {
	foo()
	caller1()
	caller2()
	f()
}

//go:noinline
func f() {
	i := 0
	p0 := &i
	p1 := &p0
	p2 := &p1
	sinki = &p2
}

//go:noinline
func foo() {
	var fooi int
	pi = &fooi
}

//go:noinline
func callee(p *int) {
	pi = p
}

//go:noinline
func caller1() {
	x := 10
	p := &x
	callee(p)
}

//go:noinline
func caller2() {
	x := 10
	callee(&x)
}
