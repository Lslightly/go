package main

import (
	"log"
	"os"
	"runtime/trace"
)

func main() {
	os.Setenv("GODEBUG", "traceallocfree=1")
	tracef, err := os.Create("alloc.trace")
	if err != nil {
		log.Fatal(err)
	}
	defer tracef.Close()
	trace.Start(tracef)
	defer trace.Stop()

	var m map[int]bool = make(map[int]bool)
	for i := range 100 {
		m[i] = true
	}
	sum := 0
	for i := 50; i < 150; i++ {
		if m[i] {
			sum += i
		}
	}
	println(sum)
}
