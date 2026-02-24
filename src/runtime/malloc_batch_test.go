//go:build linux && batch

package runtime_test

import (
	. "runtime"
	"runtime/metrics"
	"testing"
	"unsafe"
)

// sync with malloc_batchlib.go ringCap
const ringCap int8 = 8

func TestPushPop(t *testing.T) {
	fullIdx, front, end := TestRingBufferPushPop()
	if fullIdx != (ringCap-1)*2-1 || front != ringCap-2 || front != end {
		t.Fatalf("fullIdx: %d, front: %d, end: %d, ringCap: %d\n", fullIdx, front, end, ringCap)
	}
}

func BenchmarkMalloc8(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := new(int64)
		Escape(p)
	}
	BatchExit()
}

func BenchmarkMalloc16(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := new([2]int64)
		Escape(p)
	}
	BatchExit()
}

func BenchmarkMallocTypeInfo8(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := new(struct {
			p [8 / unsafe.Sizeof(uintptr(0))]*int
		})
		Escape(p)
	}
	BatchExit()
}

func BenchmarkMallocTypeInfo16(b *testing.B) {
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := new(struct {
			p [16 / unsafe.Sizeof(uintptr(0))]*int
		})
		Escape(p)
	}
	BatchExit()
}

type LargeStruct struct {
	x [16][]byte
}

var largeStructRunCnt int = 0

func BenchmarkMallocLargeStruct(b *testing.B) {
	b.ResetTimer()
	BatchEnter()
	for i := 0; i < b.N; i++ {
		p := make([]LargeStruct, 2)
		Escape(p)
	}
	BatchExit()
	b.StopTimer()
	descs := metrics.All()
	sample := make([]metrics.Sample, len(descs))
	for i, desc := range descs {
		sample[i].Name = desc.Name
	}
	metrics.Read(sample)
	const epsilon = 1e-15
	for i, desc := range descs {
		switch desc.Kind {
		case metrics.KindUint64:
			v := sample[i].Value.Uint64()
			if v == 0 {
				continue
			}
			b.ReportMetric(float64(v), desc.Name)
		case metrics.KindFloat64:
			v := sample[i].Value.Float64()
			if v < epsilon {
				continue
			}
			b.ReportMetric(v, desc.Name)
		case metrics.KindFloat64Histogram:
			v := medianBucket(sample[i].Value.Float64Histogram())
			if v < epsilon {
				continue
			}
			b.ReportMetric(v, desc.Name)
		case metrics.KindBad:
			panic("bug in runtime/metrics package!")
		}
	}
	largeStructRunCnt += b.N
	b.Log("LargeStruct Run Cnt", largeStructRunCnt)
}

func medianBucket(h *metrics.Float64Histogram) float64 {
	total := uint64(0)
	for _, count := range h.Counts {
		total += count
	}
	thresh := total / 2
	total = 0
	for i, count := range h.Counts {
		total += count
		if total >= thresh {
			return h.Buckets[i]
		}
	}
	panic("should not happen")
}

func calcSum(N int) int64 {
	sum := 0
	for i := 0; i < N; i++ {
		sum += i
	}
	return int64(sum)
}

func BenchmarkMalloc8Concurrent(b *testing.B) {
	expectedSum := calcSum(b.N)
	ch := make(chan *int64, 20)
	go func() {
		sum := int64(0)
		for p := range ch {
			sum += *p
		}
		if sum != expectedSum {
			b.Logf("expected sum: %d, got sum %d\n", expectedSum, sum)
			b.Fail()
		}
	}()
	b.ResetTimer()
	func() {
		BatchEnter()
		defer BatchExit()
		for i := 0; i < b.N; i++ {
			p := new(int64)
			*p = int64(i)
			ch <- p
			Escape(p)
		}
	}()
	b.StopTimer()
}

func BenchmarkMalloc16Concurrent(b *testing.B) {
	expectedSum := calcSum(b.N)
	ch := make(chan *[2]int64, 20)
	go func() {
		sum := int64(0)
		for p := range ch {
			sum += (*p)[0]
		}
		if sum != expectedSum {
			b.Logf("expected sum: %d, got sum %d\n", expectedSum, sum)
			b.Fail()
		}
	}()
	b.ResetTimer()
	func() {
		BatchEnter()
		defer BatchExit()
		for i := 0; i < b.N; i++ {
			p := new([2]int64)
			(*p)[0] = int64(i)
			ch <- p
			Escape(p)
		}
	}()
	b.StopTimer()
}
