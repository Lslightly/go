package main

type S struct {
	a int
}

var sink interface{}

//go:noinline
func (s *S) String() {
	pb := new(int)
	pa := &s.a
	sink = pa
	sink = pb
}

func main() {
	s := S{
		a: 2,
	}
	s.String()
}
