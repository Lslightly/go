package ret

func returnAddress() **int {
	i := 10
	p1 := &i
	p2 := &p1
	return p2
}

func returnParam(p **int) **int {
	i := 10
	*p = &i
	return p
}
