package paramleak

var sink *int

func callee(p *int) {
	sink = p
}
