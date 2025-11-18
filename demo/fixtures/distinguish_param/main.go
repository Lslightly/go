package distinguishparam

func foo() *int {
	a := 1
	bar(&a)
	return &a
}

//go:noinline
func bar(pa *int) {
	go func() {
		*pa = 1
	}()
}
