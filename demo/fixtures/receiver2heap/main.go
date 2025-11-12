package main

type S struct {
	a int
}

var sink interface{}

//go:noinline
func (s *S) String() {
	sink = s
	pa := new(int)
	sink = pa
}

func main() {
	s := S{
		a: 2,
	}
	s.String()
}
