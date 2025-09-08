// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Tests for leaks parsing with escape reasons
*/

package escape

import "testing"

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
