package indirect

func indirect() {
	var p **int
	var i int
	*p = &i
}

type S struct {
	p *int
}

func dotptr() {
	s := &S{
		p: nil,
	}
	a := 1
	s.p = &a
}

func mapidx() {
	var m map[*S]*int = make(map[*S]*int)
	a := 1
	s := &S{
		p: &a,
	}
	m[s] = &a
}
