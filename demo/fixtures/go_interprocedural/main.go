package main

import "sync"

func main() {
	a := 1 // must be marked as escaped because of Go
	var wg sync.WaitGroup
	intTransfer(&a, &wg)
	wg.Wait()
	println(a)
}

func intTransfer(px *int, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		(*px)++   // capture by value
		wg.Done() // capture by value
	}()
}
