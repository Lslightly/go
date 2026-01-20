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

/*
fullTime must be ringCap-2. [0, ..., ringCap-2] fills ring buffer
Then front becomes ringCap-1, end is ringCap-1 also.
After ringCap-1 push, end is ringCap-2, front becomes ringCap-2, fullTime is (ringCap-1)*2-1
*/
func testRingBufferPushPop() (fullIdx, front, end int8) {
	ring := &batchTransTy{}
	for i := range (ringCap - 1) * 2 {
		ring.push(fakepair(i))
		if ring.full() {
			ring.clear()
			fullIdx = i
		}
	}
	for i := ring.front; i != ring.end; i = (i + 1) % ringCap {
		FreeMSpan((*MSpan)(ring.todos[i].span))
	}
	return fullIdx, ring.front, ring.end
}

var TestRingBufferPushPop = testRingBufferPushPop
