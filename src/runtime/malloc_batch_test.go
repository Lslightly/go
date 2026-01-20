//go:build linux && batch

package runtime_test

import (
	. "runtime"
	"testing"
)

// sync with malloc_batchlib.go ringCap
const ringCap int8 = 14

func TestPushPop(t *testing.T) {
	fullIdx, front, end := TestRingBufferPushPop()
	if fullIdx != (ringCap-1)*2-1 || front != ringCap-2 || front != end {
		t.Fatalf("fullIdx: %d, front: %d, end: %d, ringCap: %d\n", fullIdx, front, end, ringCap)
	}
}
