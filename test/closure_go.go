package main

import "fmt"

func main() {
	capture_by_ref()
	capture_by_val()
}

func capture_by_ref() {
	// Capture by reference
	x := 0
	go func() {
		x++
	}()
	_ = x
}

func capture_by_val() {
	// Capture by value
	x := 0
	go func() {
		y := x + 1
		fmt.Println(y)
	}()
	_ = x
}
