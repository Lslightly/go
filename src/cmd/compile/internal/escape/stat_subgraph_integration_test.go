package escape

import (
	"fmt"
	"internal/testenv"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	gndot "gonum.org/v1/gonum/graph/encoding/dot"
)

type fixture struct {
	name     string   // fixture name
	pkgs     []string // fixture pkgs, like "main", "demo/fixtures/easyPkg/utils"
	statRoot string   // stat root dir of fixture
}

func (f fixture) allcsv() string {
	return filepath.Join(f.statRoot, f.name, "all.csv")
}

func (f fixture) graph(pkg string, batch int, phase phaseTy) string {
	return filepath.Join(f.statRoot, f.name, "graphs", pkg, strconv.Itoa(batch), phase2name[phase]+".gv")
}

func (f fixture) csv(pkg string, batch int) string {
	return filepath.Join(f.statRoot, f.name, "csv", pkg, strconv.Itoa(batch), "all.csv")
}

func goroot(t *testing.T) string {
	return testenv.GOROOT(t)
}

func demoDir(t *testing.T) string {
	return filepath.Join(goroot(t), "demo")
}

func demoStatRootDir(t *testing.T) string {
	return filepath.Join(demoDir(t), "stat_root")
}

func unmarshalEsg(graphPath string) (*escapeSubGraph, error) {
	bs, err := os.ReadFile(graphPath)
	if err != nil {
		return nil, err
	}
	esg := &escapeSubGraph{
		newid:  1,
		loc2id: make(map[*location]int64),
		nodes:  make(map[int64]*escLocNode),
		edges:  make(map[int64]map[int64]*escFlow),
		b:      nil,
	}
	err = gndot.Unmarshal(bs, esg)
	if err != nil {
		return nil, err
	}
	return esg, nil
}

func scope(t *testing.T, graphPath string, test func(esg *escapeSubGraph)) {
	esg, err := unmarshalEsg(graphPath)
	assert.Nil(t, err)
	test(esg)
}

func allNotNil[T *escLocNode | *escFlow](t *testing.T, nodes ...T) {
	for i, n := range nodes {
		assert.NotNil(t, n, fmt.Sprint("arg", i))
	}
}

func TestGoInterprocedural(t *testing.T) {
	fixture := fixture{
		name: "go_interprocedural",
		pkgs: []string{
			"main",
		},
		statRoot: demoStatRootDir(t),
	}

	// main init phase
	scope(t, fixture.graph("main", 1, pInit), func(esg *escapeSubGraph) {
		heap := esg.findNode("{heap}")
		a := esg.findNode("a @ main.go:6:2")
		wg := esg.findNode("wg @ main.go:7:6")

		allNotNil(t, heap, a, wg)
		outedges := esg.EdgeFrom(a.id)
		// a -> heap
		assert.Equal(t, 1, len(outedges))

		toHeapEdge := esg.Flow(a, heap)
		assert.NotNil(t, toHeapEdge)
		assert.Subset(t, toHeapEdge.whys, []string{FPGo, FPRootOutliveAddrSrc})

		outedges = esg.EdgeFrom(wg.id)
		// wg -> heap
		assert.Equal(t, 1, len(outedges))

		toHeapEdge = esg.Flow(wg, heap)
		assert.NotNil(t, toHeapEdge)
		assert.Subset(t, toHeapEdge.whys, []string{FPGo, FPRootOutliveAddrSrc})
	})

	// main summarize phase
	scope(t, fixture.graph("main", 1, pSummarize), func(esg *escapeSubGraph) {
		a := esg.findNode("a @ main.go:6:2")
		assert.NotNil(t, a)
		assert.True(t, a.reason.contains(ERGo|ERHeap|ERParamLeak))
		assert.False(t, a.reason.contains(EROutliveRootStack))

		wg := esg.findNode("wg @ main.go:7:6")
		assert.NotNil(t, wg)
		assert.True(t, wg.reason.contains(ERGo|ERHeap|ERParamLeak))
		assert.False(t, a.reason.contains(EROutliveRootStack))
	})
}

func TestClosureGo(t *testing.T) {
	fixture := fixture{
		name: "closure_go",
		pkgs: []string{
			"main",
		},
		statRoot: demoStatRootDir(t),
	}

	// go_capture_by_ref init phase
	scope(t, fixture.graph("main", 0, pInit), func(esg *escapeSubGraph) {
		heap := esg.findNode("{heap}")
		x := esg.findNode("x @ closure_go.go:14:2")
		closure := esg.findNode("storage for func literal @ closure_go.go:15:5")
		allNotNil(t, heap, x, closure)

		assert.False(t, x.reason.contains(ERHeap|ERGo|ERClosure))

		// x -- reference --> closure
		x2closureEdge := esg.Flow(x, closure)
		assert.NotNil(t, x2closureEdge)
		assert.Subset(t, x2closureEdge.whys, []string{"reference", FPDstEscAddrSrc})

		// closure -- go --> heap
		closure2heapEdge := esg.Flow(closure, heap)
		assert.NotNil(t, closure2heapEdge)
		assert.Subset(t, closure2heapEdge.whys, []string{FPGo})
	})

	// go_capture_by_ref removeheap phase
	scope(t, fixture.graph("main", 0, pRemoveHeap), func(esg *escapeSubGraph) {
		// heap removed
		heap := esg.findNode("{heap}")
		assert.Nil(t, heap)

		x := esg.findNode("x @ closure_go.go:14:2")
		assert.NotNil(t, x)
		assert.False(t, x.reason.contains(ERHeap|ERGo|ERClosure))

		closure := esg.findNode("storage for func literal @ closure_go.go:15:5")
		assert.NotNil(t, closure)

		assert.True(t, closure.reason.contains(ERGo|ERHeap))
		assert.True(t, x.reason.empty())
	})

	// go_capture_by_ref backPropogate phase
	scope(t, fixture.graph("main", 0, pBackPropogate), func(esg *escapeSubGraph) {
		x := esg.findNode("x @ closure_go.go:14:2")
		assert.NotNil(t, x)
		assert.True(t, x.reason.contains(ERGo|ERHeap|ERClosure))
	})

	// go_capture_by_val init phase
	scope(t, fixture.graph("main", 1, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		heap := esg.findNode("{heap}")
		closure := esg.findNode("storage for func literal @ closure_go.go:24:5")
		allNotNil(t, heap, closure)

		// closure -- go --> heap
		closure2heapEdge := esg.Flow(closure, heap)
		assert.NotNil(t, closure2heapEdge)
		assert.Subset(t, closure2heapEdge.whys, []string{FPGo})
	})

	// go_capture_by_val removeheap phase
	scope(t, fixture.graph("main", 1, pRemoveHeap), func(esg *escapeSubGraph) {
		// heap removed
		heap := esg.findNode("{heap}")
		assert.Nil(t, heap)

		closure := esg.findNode("storage for func literal @ closure_go.go:24:5")
		assert.NotNil(t, closure)

		assert.True(t, closure.reason.contains(ERGo|ERHeap))
	})

	// for_defer init phase
	scope(t, fixture.graph("main", 2, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		heap := esg.findNode("{heap}")
		y := esg.findNode("y @ closure_go.go:60:2")
		closure := esg.findNode("storage for func literal @ closure_go.go:62:9")
		allNotNil(t, heap, y, closure)

		assert.False(t, y.reason.contains(ERHeap|ERDefer))

		// y -- reference --> closure
		y2closureEdge := esg.Flow(y, closure)
		assert.NotNil(t, y2closureEdge)
		assert.Subset(t, y2closureEdge.whys, []string{"reference", FPDstEscAddrSrc})

		// closure -- defer --> heap
		closure2heapEdge := esg.Flow(closure, heap)
		assert.NotNil(t, closure2heapEdge)
		assert.Subset(t, closure2heapEdge.whys, []string{FPDefer, FPDstEscAddrSrc})
	})

	// for_defer removeheap phase
	scope(t, fixture.graph("main", 2, pRemoveHeap), func(esg *escapeSubGraph) {
		// heap removed
		heap := esg.findNode("{heap}")
		assert.Nil(t, heap)

		y := esg.findNode("y @ closure_go.go:60:2")
		assert.NotNil(t, y)
		assert.False(t, y.reason.contains(ERHeap|ERDefer))

		closure := esg.findNode("storage for func literal @ closure_go.go:62:9")
		assert.NotNil(t, closure)

		assert.True(t, closure.reason.contains(ERDefer|ERHeap))
		assert.True(t, y.reason.empty())
	})

	// for_defer backPropogate phase
	scope(t, fixture.graph("main", 2, pBackPropogate), func(esg *escapeSubGraph) {
		y := esg.findNode("y @ closure_go.go:60:2")
		assert.NotNil(t, y)
		assert.True(t, y.reason.contains(ERDefer|ERHeap|ERClosure))
	})

	// GoRoutine init phase
	scope(t, fixture.graph("main", 3, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		heap := esg.findNode("{heap}")
		x := esg.findNode("x @ closure_go.go:55:2")
		closure := esg.findNode("{storage for func literal @ closure_go.go:56:2}")
		allNotNil(t, heap, x, closure)

		assert.False(t, x.reason.contains(ERHeap|ERGo))

		// x -- address-of --> closure
		x2closureEdge := esg.Flow(x, closure)
		assert.NotNil(t, x2closureEdge)
		assert.Subset(t, x2closureEdge.whys, []string{FPRootOutliveAddrSrc, "address-of"}) // TODO closure capture &x by value, but x is not captured by value, but by reference.

		// closure -- go --> heap
		closure2heapEdge := esg.Flow(closure, heap)
		assert.NotNil(t, closure2heapEdge)
		assert.Subset(t, closure2heapEdge.whys, []string{FPGo, FPDstEscAddrSrc})
	})

	// GoRoutine removeheap phase
	scope(t, fixture.graph("main", 3, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)

		heap := esg.findNode("{heap}")
		assert.Nil(t, heap)
		x := esg.findNode("x @ closure_go.go:55:2")
		closure := esg.findNode("{storage for func literal @ closure_go.go:56:2}")

		allNotNil(t, x, closure)

		assert.False(t, x.reason.contains(ERHeap|ERGo))
		assert.True(t, closure.reason.contains(ERGo|ERHeap))
		assert.True(t, x.reason.empty())
	})

	// GoRoutine backPropogate phase
	scope(t, fixture.graph("main", 3, pBackPropogate), func(esg *escapeSubGraph) {
		x := esg.findNode("x @ closure_go.go:55:2")
		assert.NotNil(t, x)
		assert.True(t, x.reason.contains(ERGo|ERHeap))

		closure := esg.findNode("{storage for func literal @ closure_go.go:56:2}")
		assert.NotNil(t, closure)
		assert.True(t, closure.reason.contains(ERGo|ERHeap))
	})
}

func TestEasyPkg(t *testing.T) {
	fixture := fixture{
		name: "easyPkg",
		pkgs: []string{
			"main",
			"demo/fixtures/easyPkg/utils",
		},
		statRoot: demoStatRootDir(t),
	}

	// main init phase
	scope(t, fixture.graph("main", 0, pInit), func(esg *escapeSubGraph) {
		heap := esg.findNode("{heap}")
		assert.NotNil(t, heap)

		y := esg.findNode("y @ main.go:10:2")
		assert.NotNil(t, y)
		assert.False(t, y.reason.contains(ERHeap|ERParamLeak))

		// y -- address-of --> heap
		y2heapEdge := esg.Flow(y, heap)
		assert.NotNil(t, y2heapEdge)
		assert.Subset(t, y2heapEdge.whys, []string{FPRootOutliveAddrSrc, "address-of", FPToStar, FPParamLeak})
	})

	// main removeheap phase
	scope(t, fixture.graph("main", 0, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 1)

		y := esg.findNode("y @ main.go:10:2")
		assert.NotNil(t, y)
		assert.True(t, y.reason.contains(ERHeap|ERParamLeak|ERStar))
	})

	// utils Util init phase
	scope(t, fixture.graph("demo/fixtures/easyPkg/utils", 0, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)

		p := esg.findNode("p @ utils.go:4:11")
		ret0 := esg.findNode("~r0 @ utils.go:4:19")
		allNotNil(t, p, ret0)

		assert.False(t, p.reason.contains(ERReturn|ERParamLeak))
		assert.False(t, ret0.reason.contains(ERReturn|ERRootStack))

		p2ret0 := esg.Flow(p, ret0)
		assert.NotNil(t, p2ret0)
		assert.Subset(t, p2ret0.whys, []string{"return", FPParamLeak})
	})

	// utils Util removeheap phase
	scope(t, fixture.graph("demo/fixtures/easyPkg/utils", 0, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		p := esg.findNode("p @ utils.go:4:11")
		ret0 := esg.findNode("~r0 @ utils.go:4:19")
		allNotNil(t, p, ret0)

		assert.False(t, ret0.reason.contains(ERRootStack|EROutliveRootStack))
	})

	// utils Util backPropogate phase
	scope(t, fixture.graph("demo/fixtures/easyPkg/utils", 0, pBackPropogate), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		p := esg.findNode("p @ utils.go:4:11")
		assert.NotNil(t, p)
		assert.True(t, p.reason.contains(ERReturn|ERParamLeak))
		assert.False(t, p.reason.contains(ERHeap))
	})

	// utils ToHeap init phase
	scope(t, fixture.graph("demo/fixtures/easyPkg/utils", 1, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)

		px := esg.findNode("px @ utils.go:9:13")
		heap := esg.findNode("{heap}")
		allNotNil(t, px, heap)

		assert.False(t, px.reason.contains(ERStar|ERParamLeak|ERHeap))
	})

	// utils ToHeap removeheap phase
	scope(t, fixture.graph("demo/fixtures/easyPkg/utils", 1, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 1)

		px := esg.findNode("px @ utils.go:9:13")
		assert.NotNil(t, px)
		assert.True(t, px.reason.contains(ERStar|ERParamLeak|ERHeap))
	})
}

func TestIndirect(t *testing.T) {
	fixture := fixture{
		name: "indirect",
		pkgs: []string{
			"demo/fixtures/indirect",
		},
		statRoot: demoStatRootDir(t),
	}

	// indirect Indirect removeheap phase
	scope(t, fixture.graph("demo/fixtures/indirect", 0, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 1)

		i := esg.findNode("i @ indirect.go:5:6")
		assert.NotNil(t, i)

		assert.True(t, i.reason.contains(ERHeap|ERStar))
	})

	// indirect Dotptr removeheap phase
	scope(t, fixture.graph("demo/fixtures/indirect", 1, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 1)

		a := esg.findNode("a @ indirect.go:17:2")
		assert.NotNil(t, a)
		assert.True(t, a.reason.contains(ERHeap|ERStar))
	})

	// indirect Mapidx init phase
	scope(t, fixture.graph("demo/fixtures/indirect", 2, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)

		a := esg.findNode("a @ indirect.go:23:2")
		heap := esg.findNode("{heap}")
		S := esg.findNode("{storage for &S{...} @ indirect.go:24:7}")
		allNotNil(t, a, heap, S)

		assert.True(t, S.reason.contains(ERHeap)) // 3 -> 2 will set storage of &S(3) attrEscapes. When handling 1->3, if 1 is already marked as attrEscapes and 3 is attrEscapes, then reason ERHeap will be added to 3

		S2heap := esg.Flow(S, heap)
		a2heap := esg.Flow(a, heap)
		allNotNil(t, S2heap, a2heap)
		assert.Subset(t, S2heap.whys, []string{"key of map put"})

		assert.Subset(t, a2heap.whys, []string{FPToMapIndex})
	})

	// indirect Mapidx removeheap phase
	scope(t, fixture.graph("demo/fixtures/indirect", 2, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)

		a := esg.findNode("a @ indirect.go:23:2")
		S := esg.findNode("{storage for &S{...} @ indirect.go:24:7}")
		allNotNil(t, a, S)

		assert.True(t, a.reason.contains(ERHeap|ERMapIdx))
		assert.True(t, S.reason.contains(ERHeap|ERMapIdx))

		a2S := esg.Flow(a, S)
		assert.Subset(t, a2S.whys, []string{FPRootOutliveAddrSrc})
	})
}

func TestParamLeak(t *testing.T) {
	fixture := fixture{
		name: "paramleak",
		pkgs: []string{
			"demo/fixtures/paramleak",
		},
		statRoot: demoStatRootDir(t),
	}

	// paramleak callee removeheap phase
	scope(t, fixture.graph("demo/fixtures/paramleak", 0, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 1)

		p := esg.findNode("p @ paramleak.go:5:13")
		assert.NotNil(t, p)

		assert.True(t, p.reason.contains(ERHeap|ERParamLeak))
	})
}

func TestRet(t *testing.T) {
	fixture := fixture{
		name: "ret",
		pkgs: []string{
			"demo/fixtures/ret",
		},
		statRoot: demoStatRootDir(t),
	}

	// ret returnAddress init phase
	scope(t, fixture.graph("demo/fixtures/ret", 0, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		p1, r0, i := esg.findNode("p1 @ return.go:5:2"), esg.findNode("~r0 @ return.go:3:22"), esg.findNode("i @ return.go:4:2")
		allNotNil(t, p1, r0, i)

		for _, n := range []*escLocNode{p1, r0, i} {
			assert.False(t, n.reason.contains(ERHeap))
		}

		i2p1 := esg.Flow(i, p1)
		p12r0 := esg.Flow(p1, r0)
		allNotNil(t, i2p1, p12r0)

		assert.Subset(t, p12r0.whys, []string{"return"})
	})

	// ret returnAddress backPropogate phase
	scope(t, fixture.graph("demo/fixtures/ret", 0, pBackPropogate), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		p1, r0, i := esg.findNode("p1 @ return.go:5:2"), esg.findNode("~r0 @ return.go:3:22"), esg.findNode("i @ return.go:4:2")
		allNotNil(t, p1, r0, i)

		assert.True(t, p1.reason.contains(ERReturn|ERHeap))
		assert.True(t, i.reason.contains(ERReturn|ERHeap))
		assert.True(t, r0.reason.empty())
	})

	// ret returnParam init phase
	scope(t, fixture.graph("demo/fixtures/ret", 1, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 4)
		i, heap, p, r0 := esg.findNode("i @ return.go:11:2"), esg.findNode("{heap}"), esg.findNode("p @ return.go:10:18"), esg.findNode("~r0 @ return.go:10:27")
		allNotNil(t, i, heap, p, r0)

		i2heap := esg.Flow(i, heap)
		p2r0 := esg.Flow(p, r0)
		allNotNil(t, i2heap, p2r0)

		assert.Subset(t, i2heap.whys, []string{FPDstEscAddrSrc, FPToStar, "address-of"})
		assert.Subset(t, p2r0.whys, []string{FPParamLeak, "return"})
	})

	// ret returnParam backPropogate phase
	scope(t, fixture.graph("demo/fixtures/ret", 1, pBackPropogate), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		i, p, r0 := esg.findNode("i @ return.go:11:2"), esg.findNode("p @ return.go:10:18"), esg.findNode("~r0 @ return.go:10:27")
		allNotNil(t, i, p, r0)

		assert.True(t, i.reason.contains(ERHeap|ERStar))
		assert.True(t, p.reason.contains(ERParamLeak|ERReturn))
		assert.False(t, p.reason.contains(ERHeap))
		assert.True(t, r0.reason.empty())
	})
}

func TestRootstack(t *testing.T) {
	fixture := fixture{
		name: "rootstack",
		pkgs: []string{
			"demo/fixtures/rootstack",
		},
		statRoot: demoStatRootDir(t),
	}

	// rootstack outerLoopReference init phase
	scope(t, fixture.graph("demo/fixtures/rootstack", 0, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		i_testOuter, outerRef := esg.findNode("i_testOuter @ rootstack.go:7:3"), esg.findNode("outerRef @ rootstack.go:4:6")
		allNotNil(t, i_testOuter, outerRef)
		assert.False(t, i_testOuter.reason.contains(ERHeap))
		assert.False(t, outerRef.reason.contains(ERRootStack))

		i2outer := esg.Flow(i_testOuter, outerRef)
		assert.NotNil(t, i2outer)
		assert.Subset(t, i2outer.whys, []string{FPRootOutliveAddrSrc})
	})

	// rootstack outerLoopReference summarize phase
	scope(t, fixture.graph("demo/fixtures/rootstack", 0, pSummarize), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		i_testOuter, outerRef := esg.findNode("i_testOuter @ rootstack.go:7:3"), esg.findNode("outerRef @ rootstack.go:4:6")
		allNotNil(t, i_testOuter, outerRef)
		assert.True(t, i_testOuter.reason.contains(ERHeap|EROutliveRootStack))
		assert.True(t, outerRef.reason.contains(ERRootStack))
		assert.False(t, outerRef.reason.contains(EROutliveRootStack))
	})
}
