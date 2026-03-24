package main

func main() {
	s := make([]*int, 10)
	go func() {
		a := 0
		pa := &a
		s[0] = pa
		println(s[0])
	}()
}
