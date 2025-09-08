package indirect

func Indirect() {
	var p **int
	var i int
	*p = &i
}

type S struct {
	p *int
}

func Dotptr() {
	s := &S{
		p: nil,
	}
	a := 1
	s.p = &a
}

func Mapidx() {
	var m map[*S]*int = make(map[*S]*int)
	a := 1
	s := &S{
		p: &a,
	}
	m[s] = &a
}
