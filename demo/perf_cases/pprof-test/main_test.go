package main

import (
	"testing"
)

func BenchmarkFooIFace(b *testing.B) {
	for b.Loop() {
		Ifacecall()
	}
}

func BenchmarkFooGeneric(b *testing.B) {
	for b.Loop() {
		Genericcall()
	}
}

func BenchmarkAllocSync(b *testing.B) {
	for b.Loop() {
		allocSync()
	}
}

