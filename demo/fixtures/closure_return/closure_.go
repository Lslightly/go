package main

func main() {
	clo()()
}

func clo() func() {
	a := 1
	fn := func() {
		a = 1
		println(a)
	}
	return fn
}
