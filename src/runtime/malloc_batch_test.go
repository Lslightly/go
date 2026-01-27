//go:build linux && batch

package runtime_test

import (
	. "runtime"
	"testing"
	"unsafe"
)

// sync with malloc_batchlib.go ringCap
const ringCap int8 = 14

func TestPushPop(t *testing.T) {
	fullIdx, front, end := TestRingBufferPushPop()
	if fullIdx != (ringCap-1)*2-1 || front != ringCap-2 || front != end {
		t.Fatalf("fullIdx: %d, front: %d, end: %d, ringCap: %d\n", fullIdx, front, end, ringCap)
	}
}

func BenchmarkMalloc8Batch(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := new(int64)
		Escape(p)
	}
	BatchExit()
}

func BenchmarkMalloc16Batch(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := new([2]int64)
		Escape(p)
	}
	BatchExit()
}

func BenchmarkMallocTypeInfo8Batch(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := new(struct {
			p [8 / unsafe.Sizeof(uintptr(0))]*int
		})
		Escape(p)
	}
	BatchExit()
}

func BenchmarkMallocTypeInfo16Batch(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := new(struct {
			p [16 / unsafe.Sizeof(uintptr(0))]*int
		})
		Escape(p)
	}
	BatchExit()
}

func BenchmarkMallocLargeStructBatch(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := make([]LargeStruct, 2)
		Escape(p)
	}
	BatchExit()
}
