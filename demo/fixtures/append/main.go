package main

func main() {
	println(appendMany())
	println(appendDDD())
}

//go:noinline
func appendMany() []*int {
	p1 := new(int)
	p2 := new(int)
	s2 := make([]*int, 0)
	return append(s2, p1, p2)
}

//go:noinline
func appendDDD() []*int {
	p3 := new(int)
	p4 := new(int)
	s0 := []*int{p3, p4}
	s1 := []*int{new(int)}
	return append(s0, s1...)
}
