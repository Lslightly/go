package main

import "fmt"

func main() {
	go_capture_by_ref()
	go_capture_by_val()
	defer_capture_by_ref()
	defer_capture_by_val()
}

func go_capture_by_ref() {
	// Capture by reference
	x := 0
	go func() {
		x++
	}()
	_ = x
}

func go_capture_by_val() {
	// Capture by value
	x := 0
	go func() {
		y := x + 1
		fmt.Println(y)
	}()
	_ = x
}

func defer_capture_by_ref() {
	// Defer capture by reference
	x := 0
	defer func() {
		x++
	}()
	_ = x
}

func defer_capture_by_val() {
	// Defer capture by value
	x := 0
	defer func() {
		y := x + 1
		fmt.Println(y)
	}()
	_ = x
}
