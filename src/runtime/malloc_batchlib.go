//go:build linux && batch

package runtime

import (
	"internal/goarch"
	"unsafe"
)

type mallocPairsTy struct {
	span *mspan
	x    unsafe.Pointer
}

const ringCap int8 = 8

type batchTodoRingTy [ringCap]mallocPairsTy

func nextRingIdx(idx int8) int8 { return (idx + 1) & 0b111 }
func (mp *m) batchFull() bool {
	return nextRingIdx(mp.batchEnd) == mp.batchFront
}
func (mp *m) batchPush(todo mallocPairsTy) {
	mp.batchTodos[mp.batchEnd] = todo
	// println("push at", mp.batchEnd, "span addr", todo.span)
	mp.batchEnd = nextRingIdx(mp.batchEnd)
}
func (mp *m) batchClear() {
	mp.batchFront = mp.batchEnd
}

func (mp *m) batchStatusPrint() {
	println("mp batchFront", mp.batchFront, "batchEnd", mp.batchEnd, "checkGCTigger", mp.batchCheckGCTrigger)
}

func BatchEnter() {
	mp := acquirem()
	if doubleCheckMalloc {
		if mp.mallocing != 0 {
			throw("malloc deadlock")
		}
		if mp.gsignal == getg() {
			throw("malloc during signal")
		}
	}
	mp.mallocing = 1
	mp.batching = true
}

func (mp *m) batchFlush() {
	// Ensure that the stores above that initialize x to
	// type-safe memory and set the heap bits occur before
	// the caller can make x observable to the garbage
	// collector. Otherwise, on weakly ordered machines,
	// the garbage collector could follow a pointer to x,
	// but see uninitialized memory or stale heap bits.
	publicationBarrier()
	// mp.statusPrint()
	for idx := mp.batchFront; idx != mp.batchEnd; idx = nextRingIdx(idx) {
		todo := mp.batchTodos[idx]
		// println("todo idx", idx, "span addr", todo.span)

		// As x and the heap bits are initialized, update
		// freeIndexForScan now so x is seen by the GC
		// (including conservative scan) as an allocated object.
		// While this pointer can't escape into user code as a
		// _live_ pointer until we return, conservative scanning
		// may find a dead pointer that happens to point into this
		// object. Delaying this update until now ensures that
		// conservative scanning considers this pointer dead until
		// this point.
		todo.span.freeIndexForScan = todo.span.freeindex
	}
	// Allocate black during GC.
	// All slots hold nil so no scanning is needed.
	// This may be racing with GC so do it atomically if there can be
	// a race marking the bit.
	if writeBarrier.enabled {
		for idx := mp.batchFront; idx != mp.batchEnd; idx = nextRingIdx(idx) {
			todo := mp.batchTodos[idx]
			gcmarknewobject(todo.span, uintptr(todo.x))
		}
	}
	mp.batchClear()
}

func BatchExit() {
	mp := getg().m
	mp.batchFlush()
	if mp.batchCheckGCTrigger {
		if t := (gcTrigger{kind: gcTriggerHeap}); t.test() {
			gcStart(t)
		}
	}
	mp.batching = false
	mp.mallocing = 0
	releasem(mp)
}

func mallocgcTinyCore(size uintptr, typ *_type, needzero bool) (x unsafe.Pointer, elemsize uintptr, span *mspan, checkGCTrigger bool) {
	mp := getg().m
	if doubleCheckMalloc {
		if typ != nil && typ.Pointers() {
			throw("expected noscan type for noscan alloc")
		}
	}

	// Tiny allocator.
	//
	// Tiny allocator combines several tiny allocation requests
	// into a single memory block. The resulting memory block
	// is freed when all subobjects are unreachable. The subobjects
	// must be noscan (don't have pointers), this ensures that
	// the amount of potentially wasted memory is bounded.
	//
	// Size of the memory block used for combining (maxTinySize) is tunable.
	// Current setting is 16 bytes, which relates to 2x worst case memory
	// wastage (when all but one subobjects are unreachable).
	// 8 bytes would result in no wastage at all, but provides less
	// opportunities for combining.
	// 32 bytes provides more opportunities for combining,
	// but can lead to 4x worst case wastage.
	// The best case winning is 8x regardless of block size.
	//
	// Objects obtained from tiny allocator must not be freed explicitly.
	// So when an object will be freed explicitly, we ensure that
	// its size >= maxTinySize.
	//
	// SetFinalizer has a special case for objects potentially coming
	// from tiny allocator, it such case it allows to set finalizers
	// for an inner byte of a memory block.
	//
	// The main targets of tiny allocator are small strings and
	// standalone escaping variables. On a json benchmark
	// the allocator reduces number of allocations by ~12% and
	// reduces heap size by ~20%.
	c := getMCache(mp)
	off := c.tinyoffset
	// Align tiny pointer for required (conservative) alignment.
	if size&7 == 0 {
		off = alignUp(off, 8)
	} else if goarch.PtrSize == 4 && size == 12 {
		// Conservatively align 12-byte objects to 8 bytes on 32-bit
		// systems so that objects whose first field is a 64-bit
		// value is aligned to 8 bytes and does not cause a fault on
		// atomic access. See issue 37262.
		// TODO(mknyszek): Remove this workaround if/when issue 36606
		// is resolved.
		off = alignUp(off, 8)
	} else if size&3 == 0 {
		off = alignUp(off, 4)
	} else if size&1 == 0 {
		off = alignUp(off, 2)
	}
	if off+size <= maxTinySize && c.tiny != 0 {
		// The object fits into existing tiny block.
		x = unsafe.Pointer(c.tiny + off)
		c.tinyoffset = off + size
		c.tinyAllocs++
		return
	}
	// Allocate a new maxTinySize block.
	span = c.alloc[tinySpanClass]
	v := nextFreeFast(span)
	if v == 0 {
		v, span, checkGCTrigger = c.nextFree(tinySpanClass)
	}
	x = unsafe.Pointer(v)
	(*[2]uint64)(x)[0] = 0
	(*[2]uint64)(x)[1] = 0
	// See if we need to replace the existing tiny block with the new one
	// based on amount of remaining free space.
	if !raceenabled && (size < c.tinyoffset || c.tiny == 0) {
		// Note: disabled when race detector is on, see comment near end of this function.
		c.tiny = uintptr(x)
		c.tinyoffset = size
	}

	// Note cache c only valid while m acquired; see #47302
	//
	// N.B. Use the full size because that matches how the GC
	// will update the mem profile on the "free" side.
	//
	// TODO(mknyszek): We should really count the header as part
	// of gc_sys or something. The code below just pretends it is
	// internal fragmentation and matches the GC's accounting by
	// using the whole allocation slot.
	c.nextSample -= int64(span.elemsize)
	if c.nextSample < 0 || MemProfileRate != c.memProfRate {
		profilealloc(mp, x, span.elemsize)
	}

	if raceenabled {
		// Pad tinysize allocations so they are aligned with the end
		// of the tinyalloc region. This ensures that any arithmetic
		// that goes off the top end of the object will be detectable
		// by checkptr (issue 38872).
		// Note that we disable tinyalloc when raceenabled for this to work.
		// TODO: This padding is only performed when the race detector
		// is enabled. It would be nice to enable it if any package
		// was compiled with checkptr, but there's no easy way to
		// detect that (especially at compile time).
		// TODO: enable this padding for all allocations, not just
		// tinyalloc ones. It's tricky because of pointer maps.
		// Maybe just all noscan objects?
		x = add(x, span.elemsize-size)
	}
	return x, span.elemsize, span, checkGCTrigger
}

func mallocgcSmallNoscanCore(size uintptr, typ *_type, needzero bool) (unsafe.Pointer, uintptr, *mspan, bool) {
	// Set mp.mallocing to keep from being preempted by GC.
	mp := getg().m
	if doubleCheckMalloc {
		if typ != nil && typ.Pointers() {
			throw("expected noscan type for noscan alloc")
		}
	}

	checkGCTrigger := false
	c := getMCache(mp)
	var sizeclass uint8
	if size <= smallSizeMax-8 {
		sizeclass = size_to_class8[divRoundUp(size, smallSizeDiv)]
	} else {
		sizeclass = size_to_class128[divRoundUp(size-smallSizeMax, largeSizeDiv)]
	}
	size = uintptr(class_to_size[sizeclass])
	spc := makeSpanClass(sizeclass, true)
	span := c.alloc[spc]
	v := nextFreeFast(span)
	if v == 0 {
		v, span, checkGCTrigger = c.nextFree(spc)
	}
	x := unsafe.Pointer(v)
	if needzero && span.needzero != 0 {
		memclrNoHeapPointers(x, size)
	}

	// Note cache c only valid while m acquired; see #47302
	//
	// N.B. Use the full size because that matches how the GC
	// will update the mem profile on the "free" side.
	//
	// TODO(mknyszek): We should really count the header as part
	// of gc_sys or something. The code below just pretends it is
	// internal fragmentation and matches the GC's accounting by
	// using the whole allocation slot.
	c.nextSample -= int64(size)
	if c.nextSample < 0 || MemProfileRate != c.memProfRate {
		profilealloc(mp, x, size)
	}

	return x, size, span, checkGCTrigger
}

func mallocgcSmallScanNoHeaderCore(size uintptr, typ *_type, needzero bool) (unsafe.Pointer, uintptr, *mspan, bool) {
	// Set mp.mallocing to keep from being preempted by GC.
	mp := getg().m
	if doubleCheckMalloc {
		if typ == nil || !typ.Pointers() {
			throw("noscan allocated in scan-only path")
		}
		if !heapBitsInSpan(size) {
			throw("heap bits in not in span for non-header-only path")
		}
	}

	checkGCTrigger := false
	c := getMCache(mp)
	sizeclass := size_to_class8[divRoundUp(size, smallSizeDiv)]
	spc := makeSpanClass(sizeclass, false)
	span := c.alloc[spc]
	v := nextFreeFast(span)
	if v == 0 {
		v, span, checkGCTrigger = c.nextFree(spc)
	}
	x := unsafe.Pointer(v)
	if needzero && span.needzero != 0 {
		memclrNoHeapPointers(x, size)
	}
	if goarch.PtrSize == 8 && sizeclass == 1 {
		// initHeapBits already set the pointer bits for the 8-byte sizeclass
		// on 64-bit platforms.
		c.scanAlloc += 8
	} else {
		c.scanAlloc += heapSetTypeNoHeader(uintptr(x), size, typ, span)
	}
	size = uintptr(class_to_size[sizeclass])

	// Note cache c only valid while m acquired; see #47302
	//
	// N.B. Use the full size because that matches how the GC
	// will update the mem profile on the "free" side.
	//
	// TODO(mknyszek): We should really count the header as part
	// of gc_sys or something. The code below just pretends it is
	// internal fragmentation and matches the GC's accounting by
	// using the whole allocation slot.
	c.nextSample -= int64(size)
	if c.nextSample < 0 || MemProfileRate != c.memProfRate {
		profilealloc(mp, x, size)
	}

	return x, size, span, checkGCTrigger
}

func mallocgcSmallScanHeaderCore(size uintptr, typ *_type, needzero bool) (unsafe.Pointer, uintptr, *mspan, bool) {
	// Set mp.mallocing to keep from being preempted by GC.
	mp := getg().m
	if doubleCheckMalloc {
		if typ == nil || !typ.Pointers() {
			throw("noscan allocated in scan-only path")
		}
		if heapBitsInSpan(size) {
			throw("heap bits in span for header-only path")
		}
	}

	checkGCTrigger := false
	c := getMCache(mp)
	size += mallocHeaderSize
	var sizeclass uint8
	if size <= smallSizeMax-8 {
		sizeclass = size_to_class8[divRoundUp(size, smallSizeDiv)]
	} else {
		sizeclass = size_to_class128[divRoundUp(size-smallSizeMax, largeSizeDiv)]
	}
	size = uintptr(class_to_size[sizeclass])
	spc := makeSpanClass(sizeclass, false)
	span := c.alloc[spc]
	v := nextFreeFast(span)
	if v == 0 {
		v, span, checkGCTrigger = c.nextFree(spc)
	}
	x := unsafe.Pointer(v)
	if needzero && span.needzero != 0 {
		memclrNoHeapPointers(x, size)
	}
	header := (**_type)(x)
	x = add(x, mallocHeaderSize)
	c.scanAlloc += heapSetTypeSmallHeader(uintptr(x), size-mallocHeaderSize, typ, header, span)

	// Note cache c only valid while m acquired; see #47302
	//
	// N.B. Use the full size because that matches how the GC
	// will update the mem profile on the "free" side.
	//
	// TODO(mknyszek): We should really count the header as part
	// of gc_sys or something. The code below just pretends it is
	// internal fragmentation and matches the GC's accounting by
	// using the whole allocation slot.
	c.nextSample -= int64(size)
	if c.nextSample < 0 || MemProfileRate != c.memProfRate {
		profilealloc(mp, x, size)
	}
	return x, size, span, checkGCTrigger
}

func mallocgcLargeCore(size uintptr, typ *_type, needzero bool) (unsafe.Pointer, uintptr, *mspan, bool) {
	// Set mp.mallocing to keep from being preempted by GC.
	mp := getg().m

	c := getMCache(mp)
	// For large allocations, keep track of zeroed state so that
	// bulk zeroing can be happen later in a preemptible context.
	span := c.allocLarge(size, typ == nil || !typ.Pointers())
	span.freeindex = 1
	span.allocCount = 1
	span.largeType = nil // Tell the GC not to look at this yet.
	size = span.elemsize
	x := unsafe.Pointer(span.base())

	// Note cache c only valid while m acquired; see #47302
	//
	// N.B. Use the full size because that matches how the GC
	// will update the mem profile on the "free" side.
	//
	// TODO(mknyszek): We should really count the header as part
	// of gc_sys or something. The code below just pretends it is
	// internal fragmentation and matches the GC's accounting by
	// using the whole allocation slot.
	c.nextSample -= int64(size)
	if c.nextSample < 0 || MemProfileRate != c.memProfRate {
		profilealloc(mp, x, size)
	}

	// Objects can be zeroed late in a context where preemption can occur.
	// If the object contains pointers, its pointer data must be cleared
	// or otherwise indicate that the GC shouldn't scan it.
	// x will keep the memory alive.
	if noscan := typ == nil || !typ.Pointers(); !noscan || (needzero && span.needzero != 0) {
		// N.B. size == fullSize always in this case.
		memclrNoHeapPointersChunked(size, x) // This is a possible preemption point: see #47302

		if !noscan {
			getMCache(mp).scanAlloc += heapSetTypeLarge(uintptr(x), size, typ, span)
		}
	}
	return x, size, span, false
}
