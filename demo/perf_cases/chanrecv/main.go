package main

import (
	"reflect"
)

func setsc[T any](ch chan T) {
	chVal := reflect.ValueOf(ch)
	_ = chVal.UnsafePointer()
	// runtime.SCChan(chVal.UnsafePointer())
}

func main() {
	ch := make(chan int, 2)
	setsc(ch)
	go func() {
		ch <- 2
	}()
	v, ok := <-ch
	if ok {
		print(v)
	}
}
