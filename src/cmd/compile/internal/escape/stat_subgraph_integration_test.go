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

func TestEscape(t *testing.T) {
	fixt := fixture{
		name: "escape",
		pkgs: []string{
			"main",
		},
		statRoot: demoStatRootDir(t),
	}

	// escape L17 removeheap phase
	scope(t, fixt.graph("main", 0, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 1)
		makeslice := esg.findNode("{storage for make([]*int, 0, 100) @ escape.go:17:18}")
		assert.NotNil(t, makeslice)
		assert.True(t, makeslice.reason.contains(ERHeap))
	})

	// escape noalias removeheap phase
	scope(t, fixt.graph("main", 1, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		p, q := esg.findNode("p @ escape.go:19:14"), esg.findNode("q @ escape.go:19:17")
		allNotNil(t, p, q)
		assert.Nil(t, esg.findNode("{heap}"))

		assert.True(t, p.reason.contains(ERHeap|ERParamLeak|ERSliceIdx))
		assert.True(t, q.reason.contains(ERHeap|ERParamLeak|ERSliceIdx))
	})

	// escape chk init phase
	scope(t, fixt.graph("main", 2, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		p := esg.findNode("p @ escape.go:47:10")
		q := esg.findNode("q @ escape.go:47:13")
		heap := esg.findNode("{heap}")
		allNotNil(t, p, q, heap)

		p2heap := esg.Flow(p, heap)
		q2heap := esg.Flow(q, heap)
		// FIXME: FPDstEscAddrSrc and FPRootOutliveAddrSrc should not appear
		assert.Subset(t, p2heap.whys, []string{FPParamLeak, FPToIndex, FPDstEscAddrSrc, FPRootOutliveAddrSrc}) // FPDstEscAddrSrc and FPRootOutliveAddrSrc are caused by why2reason reverse mapping in towhys() method
		assert.Subset(t, q2heap.whys, []string{FPParamLeak, FPToIndex, FPDstEscAddrSrc, FPRootOutliveAddrSrc}) // FPDstEscAddrSrc and FPRootOutliveAddrSrc are caused by why2reason reverse mapping in towhys() method
	})

	// escape chk removeheap phase
	scope(t, fixt.graph("main", 2, pRemoveHeap), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		p := esg.findNode("p @ escape.go:47:10")
		q := esg.findNode("q @ escape.go:47:13")
		allNotNil(t, p, q)

		assert.True(t, p.reason.contains(ERParamLeak|ERHeap|ERSliceIdx))
		assert.True(t, q.reason.contains(ERParamLeak|ERHeap|ERSliceIdx))

		assert.Len(t, esg.EdgeFrom(p.id), 0)
		assert.Len(t, esg.EdgeFrom(q.id), 0)
	})

	// escape i_escapes summarize phase
	scope(t, fixt.graph("main", 3, pSummarize), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		i := esg.findNode("i @ escape.go:64:6")
		r0 := esg.findNode("~r0 @ escape.go:63:23")
		allNotNil(t, i, r0)

		assert.True(t, i.reason.contains(ERReturn|ERHeap))
		assert.True(t, r0.reason.empty())

		i2r0 := esg.Flow(i, r0)
		assert.NotNil(t, i2r0)
		assert.Subset(t, i2r0.whys, []string{FPRootOutliveAddrSrc, "address-of", "return"})
	})

	// escape in_escapes summarize phase
	scope(t, fixt.graph("main", 6, pSummarize), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		x := esg.findNode("x @ escape.go:80:17")
		r0 := esg.findNode("~r0 @ escape.go:80:24")
		allNotNil(t, x, r0)

		x2x := esg.Flow(x, x)
		assert.Nil(t, x2x) // there won't be x -> x edge
		x2r0 := esg.Flow(x, r0)
		assert.NotNil(t, x2r0)
		assert.Subset(t, x2r0.whys, []string{FPRootOutliveAddrSrc, FPParamLeak, "address-of", "return"})
	})

	// escape select_escapes init phase
	scope(t, fixt.graph("main", 7, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 4)
		c := esg.findNode("{storage for func literal @ escape.go:90:2}")
		heap := esg.findNode("{heap}")
		req := esg.findNode("req @ escape.go:92:7")
		ret0 := esg.findNode("~r0 @ escape.go:88:28")
		allNotNil(t, c, heap, req, ret0)

		c2heap := esg.Flow(c, heap)
		req2ret0 := esg.Flow(req, ret0)
		allNotNil(t, c2heap, req2ret0)

		assert.Subset(t, c2heap.whys, []string{FPDstEscAddrSrc, FPGo})
		assert.Subset(t, req2ret0.whys, []string{FPRootOutliveAddrSrc, "address-of", "return"})
	})

	// escape select_escapes summarize phase
	scope(t, fixt.graph("main", 7, pSummarize), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		c := esg.findNode("{storage for func literal @ escape.go:90:2}")
		req := esg.findNode("req @ escape.go:92:7")
		ret0 := esg.findNode("~r0 @ escape.go:88:28")
		allNotNil(t, c, req, ret0)
		assert.True(t, c.reason.contains(ERGo|ERHeap))
		assert.True(t, req.reason.contains(ERReturn|ERHeap))
	})

	// escape select_escapes1 init phase
	scope(t, fixt.graph("main", 8, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 6)
		closure := esg.findNode("{storage for func literal @ escape.go:105:3}")
		heap := esg.findNode("{heap}")
		req := esg.findNode("req @ escape.go:107:8")
		p := esg.findNode("p @ escape.go:101:6")
		ret1 := esg.findNode("~r1 @ escape.go:98:43")
		ret0 := esg.findNode("~r0 @ escape.go:98:37")
		allNotNil(t, closure, heap, req, p, ret1, ret0)

		closure2heap := esg.Flow(closure, heap)
		req2p := esg.Flow(req, p)
		req2ret1 := esg.Flow(req, ret1)
		req2ret0 := esg.Flow(req, ret0)
		allNotNil(t, closure2heap, req2p, req2ret1, req2ret0)

		assert.Subset(t, closure2heap.whys, []string{FPGo, FPDstEscAddrSrc})
		assert.Subset(t, req2p.whys, []string{FPRootOutliveAddrSrc, "address-of"})
		assert.Subset(t, req2ret1.whys, []string{FPRootOutliveAddrSrc, "address-of", "return"})
		assert.Subset(t, req2ret0.whys, []string{FPRootOutliveAddrSrc, "address-of", "return"})
	})

	// escape select_escapes1 summarize phase
	scope(t, fixt.graph("main", 8, pSummarize), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 5)
		closure := esg.findNode("{storage for func literal @ escape.go:105:3}")
		req := esg.findNode("req @ escape.go:107:8")
		p := esg.findNode("p @ escape.go:101:6")
		ret1 := esg.findNode("~r1 @ escape.go:98:43")
		ret0 := esg.findNode("~r0 @ escape.go:98:37")
		allNotNil(t, closure, req, p, ret1, ret0)

		assert.True(t, closure.reason.contains(ERGo|ERHeap))
		assert.True(t, req.reason.contains(EROutliveRootStack|ERHeap|ERReturn))
		assert.True(t, p.reason.contains(ERRootStack))
	})

	// escape range_escapes init phase
	scope(t, fixt.graph("main", 9, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		v := esg.findNode("v @ escape.go:117:9")
		ret0 := esg.findNode("~r0 @ escape.go:114:27")
		allNotNil(t, v, ret0)

		v2ret0 := esg.Flow(v, ret0)
		assert.Subset(t, v2ret0.whys, []string{FPRootOutliveAddrSrc, "address-of", "return"})
	})

	// escape range_escapes backPropogate phase
	scope(t, fixt.graph("main", 9, pBackPropogate), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		v := esg.findNode("v @ escape.go:117:9")
		assert.NotNil(t, v)
		assert.True(t, v.reason.contains(ERHeap|ERReturn))
	})

	// escape range_escape2 init phase
	scope(t, fixt.graph("main", 10, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		v := esg.findNode("v @ escape.go:129:9")
		ret1 := esg.findNode("~r1 @ escape.go:124:38")
		ret0 := esg.findNode("~r0 @ escape.go:124:32")
		allNotNil(t, v, ret1, ret0)

		v2r1 := esg.Flow(v, ret1)
		v2r0 := esg.Flow(v, ret0)
		allNotNil(t, v2r1, v2r0)
		assert.Subset(t, v2r1.whys, []string{"return", FPRootOutliveAddrSrc, "address-of"})
		assert.Subset(t, v2r0.whys, []string{"return", FPRootOutliveAddrSrc, "address-of"})
	})

	// escape range_escape2 summarize phase
	scope(t, fixt.graph("main", 10, pSummarize), func(esg *escapeSubGraph) {
		v := esg.findNode("v @ escape.go:129:9")
		assert.True(t, v.reason.contains(ERReturn|ERHeap))
	})

	// escape for_escapes2 init phase
	scope(t, fixt.graph("main", 11, pInit), func(esg *escapeSubGraph) {
		// TODO: add reason for outlive because of loop depth in edge
		assert.Len(t, esg.nodes, 3)
		i := esg.findNode("i @ escape.go:140:2")
		r1 := esg.findNode("~r1 @ escape.go:137:40")
		r0 := esg.findNode("~r0 @ escape.go:137:34")
		allNotNil(t, i, r1, r0)

		v2r1 := esg.Flow(i, r1)
		v2r0 := esg.Flow(i, r0)
		allNotNil(t, v2r1, v2r0)
		assert.Subset(t, v2r1.whys, []string{"return", FPRootOutliveAddrSrc, "address-of"})
		assert.Subset(t, v2r0.whys, []string{"return", FPRootOutliveAddrSrc, "address-of"})
	})

	// escape for_escapes2 summarize phase
	scope(t, fixt.graph("main", 11, pSummarize), func(esg *escapeSubGraph) {
		i := esg.findNode("i @ escape.go:140:2")
		assert.True(t, i.reason.contains(ERHeap|ERReturn))
	})

	// escape for_escapes3 init phase
	scope(t, fixt.graph("main", 12, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 4)
		newint := esg.findNode("{storage for new(int) @ escape.go:152:11}")
		r0 := esg.findNode("~r0 @ escape.go:154:17")
		closure := esg.findNode("{storage for func literal @ escape.go:154:10}")
		f := esg.findNode("f @ escape.go:149:6")
		allNotNil(t, newint, r0, closure, f)

		newint2r0 := esg.Flow(newint, r0)
		newint2closure := esg.Flow(newint, closure)
		clo2f := esg.Flow(closure, f)
		allNotNil(t, newint2r0, newint2closure, clo2f)

		assert.Subset(t, newint2r0.whys, []string{FPRootOutliveAddrSrc, "return"})
		assert.Subset(t, newint2closure.whys, []string{"by value", "captured by a closure"})
		assert.NotContains(t, newint2closure.whys, []string{"reference"})
		assert.Subset(t, clo2f.whys, []string{FPRootOutliveAddrSrc})
	})

	// escape for_escapes3 summarize phase
	scope(t, fixt.graph("main", 12, pSummarize), func(esg *escapeSubGraph) {
		newint := esg.findNode("{storage for new(int) @ escape.go:152:11}")
		r0 := esg.findNode("~r0 @ escape.go:154:17")
		closure := esg.findNode("{storage for func literal @ escape.go:154:10}")
		f := esg.findNode("f @ escape.go:149:6")
		allNotNil(t, newint, r0, closure, f)

		assert.True(t, newint.reason.contains(EROutliveRootStack|ERHeap|ERReturn))
		assert.True(t, closure.reason.contains(ERHeap|EROutliveRootStack))
		assert.True(t, f.reason.contains(ERRootStack))
		assert.False(t, f.reason.contains(EROutliveRootStack))
	})

	// escape out_escapes init phase
	scope(t, fixt.graph("main", 13, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		x := esg.findNode("x @ escape.go:160:26")
		p := esg.findNode("p @ escape.go:160:33")
		allNotNil(t, x, p)

		x2p := esg.Flow(x, p)
		assert.Subset(t, x2p.whys, []string{FPRootOutliveAddrSrc, "address-of"})
		assert.NotContains(t, x2p.whys, "return") // because `p = &x` is not `return &`, so "return" won't be added to the note of hole from x to p
	})

	// escape out_escapes summarize phase
	scope(t, fixt.graph("main", 13, pBackPropogate), func(esg *escapeSubGraph) {
		x := esg.findNode("x @ escape.go:160:26")
		assert.True(t, x.reason.contains(ERHeap|ERReturn)) // here because p.Name is ir.PARAMOUT, ERReturn will be added to x when backPropogate
	})

	// escape out_escapes_2 init phase
	scope(t, fixt.graph("main", 14, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 2)
		x := esg.findNode("x @ escape.go:166:28")
		p := esg.findNode("p @ escape.go:166:35")
		allNotNil(t, x, p)

		x2p := esg.Flow(x, p)
		assert.Subset(t, x2p.whys, []string{FPRootOutliveAddrSrc, "address-of", "return"}) // because "return &x", "return" is added as why of hole from x to p
	})

	// escape defer1 init phase
	scope(t, fixt.graph("main", 15, pInit), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 4)
		clo := esg.findNode("{storage for func literal @ escape.go:173:5}")
		heap := esg.findNode("{heap}")
		x := esg.findNode("x @ escape.go:171:21")
		i := esg.findNode("i @ escape.go:171:13")
		allNotNil(t, clo, heap, x, i)

		clo2heap := esg.Flow(clo, heap)
		x2clo := esg.Flow(x, clo)
		i2clo := esg.Flow(i, clo)
		assert.Subset(t, clo2heap.whys, []string{FPDstEscAddrSrc, FPGo})

		assert.Subset(t, x2clo.whys, []string{FPDstEscAddrSrc, "captured by a closure", "reference"})
		assert.NotContains(t, x2clo.whys, "by value")

		assert.Subset(t, i2clo.whys, []string{FPParamLeak, "captured by a closure", "by value"})
		assert.NotContains(t, i2clo.whys, []string{"reference"})
	})

	// escape defer1 summarize phase
	scope(t, fixt.graph("main", 15, pSummarize), func(esg *escapeSubGraph) {
		assert.Len(t, esg.nodes, 3)
		clo := esg.findNode("{storage for func literal @ escape.go:173:5}")
		x := esg.findNode("x @ escape.go:171:21")
		i := esg.findNode("i @ escape.go:171:13")
		allNotNil(t, clo, x, i)

		assert.True(t, clo.reason.contains(ERHeap|ERGo))
		assert.True(t, x.reason.contains(ERClosure|ERGo|ERHeap))
		assert.True(t, i.reason.contains(ERParamLeak|ERGo|ERHeap))
	})

	// escape main init phase
	scope(t, fixt.graph("main", 16, pInit), func(esg *escapeSubGraph) {
		v2 := esg.findNode("v @ escape.go:197:41") // 15 out nodes
		v1 := esg.findNode("v @ escape.go:197:22") // 15 out nodes
		assert.Len(t, esg.EdgeFrom(v2.id), 15)
		assert.Len(t, esg.EdgeFrom(v1.id), 15)
		heap := esg.findNode("{heap}")
		assert.Len(t, esg.EdgeTo(heap.id), 17)
	})

	// escape main summarize phase
	scope(t, fixt.graph("main", 16, pSummarize), func(esg *escapeSubGraph) {
		strNode := esg.findNode("{storage for \"BUG: no escape\" @ escape.go:224:9}")
		assert.True(t, strNode.reason.contains(ERHeap))
		nodesIter := esg.Nodes()
		tmpNodeSet := make(map[*escLocNode]bool)
		for nodesIter.Next() {
			n := nodesIter.Node().(*escLocNode)
			if n.IsParsedTmp() {
				// these tmp nodes are all temporary nodes created in teeHole, they are allocated in stack, but their values may escape to heap.
				assert.True(t, n.reason.contains(ERRootStack), n.label)
				assert.Equal(t, 1, n.reason.count(), n.label)
				tmpNodeSet[n] = true
			}
		}
		assert.Len(t, tmpNodeSet, 20) // 14*2 - 4*2, p, q, ~r0, p should be removed
		nodesIter.Reset()
		func() {
			// inline 1
			type nodecase struct {
				name        string
				hasSliceIdx bool
			}
			cases := []nodecase{
				{"x @ escape.go:214:22", true}, // because of chk
				{"x @ escape.go:210:20", true}, // because of chk
				{"i @ escape.go:203:21", true}, // because of p, q are passed to chk
				{"v @ escape.go:200:23", true}, // because of p, q are passed to chk
				{"x @ escape.go:188:34", true}, // because of chk
				{"k @ escape.go:185:32", true}, // because of chk
				{"j @ escape.go:182:32", true}, // because of chk
				{"i @ escape.go:179:33", true}, // because of chk
			}
			for _, nc := range cases {
				node := esg.findNode(nc.name)
				if nc.hasSliceIdx {
					assert.True(t, node.reason.contains(ERSliceIdx), node.label)
				} else {
					assert.False(t, node.reason.contains(ERSliceIdx), node.label)
				}
			}
		}()
		v2 := esg.findNode("v @ escape.go:197:41")
		v1 := esg.findNode("v @ escape.go:197:22")
		assert.True(t, v1.reason.contains(ERParamLeak|ERSliceIdx|ERHeap|EROutliveRootStack))
		assert.True(t, v2.reason.contains(ERParamLeak|ERSliceIdx|ERHeap|EROutliveRootStack))
	})
}

func TestEscape2(t *testing.T) {
	fixt := fixture{
		name: "escape2",
		pkgs: []string{
			"p",
		},
		statRoot: demoStatRootDir(t),
	}

	// escape2 foo132 summarize phase
	scope(t, fixt.graph("p", 124, pSummarize), func(esg *escapeSubGraph) {
		i := esg.findNode("i @ escape2.go:1257")
		assert.True(t, i.reason.contains(ERGo|ERClosure|ERHeap))
	})

	// escape2 foo137 summarize phase
	scope(t, fixt.graph("p", 128, pSummarize), func(esg *escapeSubGraph) {
		i := esg.findNode("i @ escape2.go:1307")
		assert.True(t, i.reason.contains(ERGo|ERClosure|ERHeap))
	})
}

func TestGlobal(t *testing.T) {
	fixt := fixture{
		name: "global",
		pkgs: []string{
			"main",
		},
		statRoot: demoStatRootDir(t),
	}

	assertContainGlobalParamCause := func(node *escLocNode) {
		assert.True(t, node.reason.contains(ERGlobal|ERParamLeak))
	}

	// main foo summarize phase
	scope(t, fixt.graph("main", 0, pSummarize), func(esg *escapeSubGraph) {
		fooi := esg.findNode("fooi @ main.go:24:6")
		assert.NotNil(t, fooi)
		assert.True(t, fooi.reason.contains(ERGlobal|ERHeap))
	})

	// main callee summarize phase
	scope(t, fixt.graph("main", 1, pSummarize), func(esg *escapeSubGraph) {
		p := esg.findNode("p @ main.go:29:13")
		assert.NotNil(t, p)
		assertContainGlobalParamCause(p)
	})

	// main caller1 summarize phase
	scope(t, fixt.graph("main", 2, pSummarize), func(esg *escapeSubGraph) {
		x := esg.findNode("x @ main.go:35:2")
		assert.NotNil(t, x)
		assertContainGlobalParamCause(x)
	})

	// main caller2 summarize phase
	scope(t, fixt.graph("main", 3, pSummarize), func(esg *escapeSubGraph) {
		x := esg.findNode("x @ main.go:42:2")
		assert.NotNil(t, x)
		assertContainGlobalParamCause(x)
	})

	// main f summarize phase
	scope(t, fixt.graph("main", 4, pSummarize), func(esg *escapeSubGraph) {
		for _, name := range []string{
			"p2 @ main.go:18:2",
			"p1 @ main.go:17:2",
			"p0 @ main.go:16:2",
			"i @ main.go:15:2",
		} {
			n := esg.findNode(name)
			assert.NotNil(t, n)
			assert.True(t, n.reason.contains(ERGlobal|ERHeap))
		}
	})
}
