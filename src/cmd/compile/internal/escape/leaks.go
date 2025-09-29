// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package escape

import (
	"cmd/compile/internal/base"
	"fmt"
	"math"
	"strings"
)

// A leaks represents a set of assignment flows from a parameter to
// the heap, mutator, callee, or to any of its function's (first
// numEscResults) result parameters.
// The first 2 uint8 is the escape reason in stat_subgraph.go
type leaks [2 + 8]uint8

const (
	leakReason0 = iota
	leakReason1
	leakHeap
	leakMutator
	leakCallee
	leakResult0
)

const numEscResults = len(leaks{}) - leakResult0

// Heap returns the minimum deref count of any assignment flow from l
// to the heap. If no such flows exist, Heap returns -1.
func (l leaks) Heap() int { return l.get(leakHeap) }

// Mutator returns the minimum deref count of any assignment flow from
// l to the pointer operand of an indirect assignment statement. If no
// such flows exist, Mutator returns -1.
func (l leaks) Mutator() int { return l.get(leakMutator) }

// Callee returns the minimum deref count of any assignment flow from
// l to the callee operand of call expression. If no such flows exist,
// Callee returns -1.
func (l leaks) Callee() int { return l.get(leakCallee) }

// Result returns the minimum deref count of any assignment flow from
// l to its function's i'th result parameter. If no such flows exist,
// Result returns -1.
func (l leaks) Result(i int) int { return l.get(leakResult0 + i) }

// AddHeap adds an assignment flow from l to the heap.
func (l *leaks) AddHeap(derefs int) { l.add(leakHeap, derefs) }

// AddMutator adds a flow from l to the mutator (i.e., a pointer
// operand of an indirect assignment statement).
func (l *leaks) AddMutator(derefs int) { l.add(leakMutator, derefs) }

// AddCallee adds an assignment flow from l to the callee operand of a
// call expression.
func (l *leaks) AddCallee(derefs int) { l.add(leakCallee, derefs) }

// AddResult adds an assignment flow from l to its function's i'th
// result parameter.
func (l *leaks) AddResult(i, derefs int) { l.add(leakResult0+i, derefs) }

func (l leaks) get(i int) int { return int(l[i]) - 1 }

func (l *leaks) add(i, derefs int) {
	if old := l.get(i); old < 0 || derefs < old {
		l.set(i, derefs)
	}
}

func (l *leaks) set(i, derefs int) {
	v := derefs + 1
	if v < 0 {
		base.Fatalf("invalid derefs count: %v", derefs)
	}
	if v > math.MaxUint8 {
		v = math.MaxUint8
	}

	l[i] = uint8(v)
}

func (l *leaks) encodeReason(er escapeReason) {
	if er < 0 || er >= math.MaxUint16 {
		base.Fatalf("invalid escape reason: %v", er)
	}
	v := uint16(er)
	l[0] = uint8(v)
	l[1] = uint8(v >> 8)
}

func (l *leaks) decodeReason() escapeReason {
	// Decode the escape reason from the first two bytes.
	// The escape reason is encoded as a 2-byte uint16.
	return escapeReason(uint16(l[0]) | uint16(l[1])<<8)
}

// Optimize removes result flow paths that are equal in length or
// longer than the shortest heap flow path.
func (l *leaks) Optimize() {
	// If we have a path to the heap, then there's no use in
	// keeping equal or longer paths elsewhere.
	if x := l.Heap(); x >= 0 {
		for i := leakHeap + 1; i < len(*l); i++ {
			if l.get(i) >= x {
				l.set(i, -1)
			}
		}
	}
}

var leakTagCache = map[leaks]string{}

// Encode converts l into a binary string for export data.
func (l leaks) Encode() string {
	if l.Heap() == 0 {
		// Space optimization: empty string encodes more
		// efficiently in export data.
		if l[leakReason0] == 0 && l[leakReason1] == 0 {
			// Space optimization: zero reason and zero esc
			return ""
		}
		// New for escape reason statistic, the first two bits should be encoded if reason is not empty
		return string(l[:leakHeap])
	}
	if s, ok := leakTagCache[l]; ok {
		return s
	}

	n := len(l)
	for n > 0 && l[n-1] == 0 {
		n--
	}
	s := string(l[:leakHeap]) + "esc:" + string(l[min(leakHeap, n):n]) // TODO: use [leakHeap:max(leakHeap, n)]
	leakTagCache[l] = s
	return s
}

// parseLeaks parses a binary string representing a leaks.
func parseLeaks(s string) leaks {
	var l leaks
	if isEAReasonStatEnabled() && isEAReasonDebug() {
		str := "parseLeaks: "
		if !strings.HasPrefix(s, "esc:") {
			if len(s) == leakHeap {
				// reason0 reason1
				str += "heap, reason: "
				str += fmt.Sprintf("0b%08b%08b", s[1], s[0])
				getLogger().Println(str)
			} else if len(s) == 0 {
				// ""
				str += "heap, no reason"
				getLogger().Println(str)
			} else {
				// reason0 reason1 esc:...
				str += "reason: "
				str += fmt.Sprintf("0b%08b%08b", s[1], s[0])
				str += " esc:"
				for i := 4 + leakHeap; i < len(s); i++ {
					str += fmt.Sprintf(" %d", s[i])
				}
				getLogger().Println(str)
			}
		} else {
			// esc:...
			str += "no reason, esc:"
			for i := 4; i < len(s); i++ {
				str += fmt.Sprintf(" %d", s[i])
			}
			getLogger().Println(str)
		}
	}
	// don't use s == "" here because the len(s) can be non-zero while s is ""
	if !strings.HasPrefix(s, "esc:") && (len(s) == leakHeap || len(s) == 0) {
		// Two cases: 1. reason+empty string. 2. empty string
		l.AddHeap(0)
		if len(s) == leakHeap {
			// reason + empty string
			copy(l[:leakHeap], s[:leakHeap]) // copy Reasons
		}
		return l
	}
	if !strings.HasPrefix(s, "esc:") {
		// reason + esc:
		copy(l[:leakHeap], s[:leakHeap]) // copy Reasons
		s = s[leakHeap:]
	}
	// esc:...
	copy(l[leakHeap:], s[4:])
	return l
}
