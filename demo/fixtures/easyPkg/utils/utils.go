package utils

//go:noinline
func Util(p *int) *int {
	return p
}

//go:noinline
func ToHeap(px *int) {
	ppx := &px
	*ppx = px
}
