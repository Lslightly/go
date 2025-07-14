package main

import (
	"strings"
	"testing"
)

// A leaks represents a set of assignment flows from a parameter to
// the heap, mutator, callee, or to any of its function's (first
// numEscResults) result parameters.
// The first 2 uint8 is the escape reason in stat_subgraph.go
type leaks [2 + 8]uint8

func (l *leaks) AddHeap(derefs int) { l[leakHeap] = uint8(derefs) + 1 }

const (
	leakReason0 = iota
	leakReason1
	leakHeap
	leakMutator
	leakCallee
	leakResult0
)

func parseLeaks(s string) leaks {
	var l leaks
	// don't use s == "" here
	if !strings.HasPrefix(s, "esc:") && (len(s) == leakHeap || len(s) == 0) {
		l.AddHeap(0)
		if len(s) == leakHeap {
			copy(l[:leakHeap], s[:leakHeap]) // copy Reasons
		}
		return l
	}
	if !strings.HasPrefix(s, "esc:") {
		copy(l[:leakHeap], s[:leakHeap]) // copy Reasons
		s = s[leakHeap:]
	}
	copy(l[leakHeap:], s[4:])
	return l
}

func TestEmpty(t *testing.T) {
	l := parseLeaks("")
	if l[0] != 0 || l[1] != 0 {
		t.Errorf("expected empty leaks, got %v", l)
	}
	if l[leakHeap] != 1 || l[leakMutator] != 0 {
		t.Errorf("expected heap 1, mutator 0, got %v", l[leakHeap:leakMutator+1])
	}
}

func TestReasonEmpty(t *testing.T) {
	l := parseLeaks(string([]byte{1, 2}))
	if l[0] != 1 || l[1] != 2 {
		t.Errorf("expected reason 1,2, got %v", l)
	}
	if l[leakHeap] != 1 || l[leakMutator] != 0 {
		t.Errorf("expected heap 1, mutator 0, got %v", l[leakHeap:leakMutator+1])
	}
}

func TestEsc(t *testing.T) {
	l := parseLeaks("esc:" + string([]byte{2, 0, 3}))
	if l[0] != 0 || l[1] != 0 {
		t.Errorf("expected reason 0,0, got %v", l)
	}
	if l[leakHeap] != 2 || l[leakMutator] != 0 {
		t.Errorf("expected heap 1, mutator 0, got %v", l[leakHeap:leakMutator+1])
	}
	if l[leakCallee] != 3 {
		t.Errorf("expected callee 3, got %v", l[leakCallee])
	}
}

func TestReasonEsc(t *testing.T) {
	l := parseLeaks(string([]byte{1, 2}) + "esc:" + string([]byte{3, 4}))
	if l[0] != 1 || l[1] != 2 {
		t.Errorf("expected reason 1,2, got %v", l)
	}
	if l[leakHeap] != 3 || l[leakMutator] != 4 {
		t.Errorf("expected heap 3, mutator 4, got %v", l[leakHeap:leakMutator+1])
	}
}

func TestReasonEscReasonEmpty(t *testing.T) {
	l := parseLeaks(string([]byte{0, 0}) + "esc:" + string([]byte{3, 4}))
	if l[0] != 0 || l[1] != 0 {
		t.Errorf("expected reason 0,0, got %v", l)
	}
	if l[leakHeap] != 3 || l[leakMutator] != 4 {
		t.Errorf("expected heap 3, mutator 4, got %v", l[leakHeap:leakMutator+1])
	}
}
