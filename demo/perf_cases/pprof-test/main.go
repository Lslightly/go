package main

import (
	"io"
	"sync"
)

type DummyWriter struct {
	buf []byte
}

func (d *DummyWriter) Write(data []byte) (int, error) {
	return 0, nil
}

func FooIFace(w io.Writer, data byte) {
	w.Write([]byte{data})
}

func FooGeneric[T io.Writer](w T, data byte) {
	w.Write([]byte{data})
}

func Ifacecall() {
	w := &DummyWriter{}
	FooIFace(w, 42)
}

func Genericcall() {
	w := &DummyWriter{}
	FooGeneric(w, 42)
}

func allocSync() {
	const MaxCoreNum int = 64
	const ElemNum int = 128
	var wg sync.WaitGroup
	for i := 0; i < MaxCoreNum; i++ {
		s := make([]int, ElemNum)
		wg.Add(1)
		go func() {
			defer wg.Done()
			sum := 0
			for j := 0; j < ElemNum; j++ {
				s[j] = j
				sum += j
			}
			println(sum)
		}()
	}
	wg.Wait()
}

func main() {
	w := &DummyWriter{}
	FooIFace(w, 42)
	FooGeneric(w, 42)
}

