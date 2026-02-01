//go:build linux && batch

package runtime

import "unsafe"

var s *mspan

func fakespan(idx uint16) *mspan {
	s := AllocMSpan()
	s.freeindex = idx
	s.freeIndexForScan = idx - 1
	return (*mspan)(s)
}

func fakepair(idx int8) mallocPairsTy {
	a := new(int)
	return mallocPairsTy{
		span: fakespan(uint16(idx)),
		x:    unsafe.Pointer(a),
	}
}

func fakem() *m {
	return &m{}
}

/*
fullTime must be ringCap-2. [0, ..., ringCap-2] fills ring buffer
Then front becomes ringCap-1, end is ringCap-1 also.
After ringCap-1 push, end is ringCap-2, front becomes ringCap-2, fullTime is (ringCap-1)*2-1
*/
func testRingBufferPushPop() (fullIdx, front, end int8) {
	mp := fakem()
	for i := range (ringCap - 1) * 2 {
		mp.batchPush(fakepair(i))
		if mp.batchFull() {
			mp.batchClear()
			fullIdx = i
		}
	}
	for i := mp.batchFront; i != mp.batchEnd; i = nextRingIdx(i) {
		FreeMSpan((*MSpan)(mp.batchTodos[i].span))
	}
	return fullIdx, mp.batchFront, mp.batchEnd
}

var TestRingBufferPushPop = testRingBufferPushPop
