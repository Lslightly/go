package indirect

func indirect() {
	var p **int
	var i int
	*p = &i
}
