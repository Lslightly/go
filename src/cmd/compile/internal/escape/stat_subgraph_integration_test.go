package escape

import (
	"internal/testenv"
	"os"
	"path/filepath"
	"slices"
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

func TestGoInterprocedural(t *testing.T) {
	fixture := fixture{
		name: "go_interprocedural",
		pkgs: []string{
			"main",
		},
		statRoot: demoStatRootDir(t),
	}

	scope(t, fixture.graph("main", 1, pInit), func(esg *escapeSubGraph) {
		heap := esg.findNode("{heap}")
		assert.NotNil(t, heap)
		a := esg.findNode("a @ main.go:6:2")
		assert.NotNil(t, a)
		outedges := esg.EdgeFrom(a.id)
		assert.Equal(t, 2, len(outedges))

		toHeapEdge, ok := esg.Edge(a.ID(), heap.ID()).(*escFlow)
		assert.NotNil(t, toHeapEdge)
		assert.True(t, ok)
		assert.Subset(t, toHeapEdge.whys, []string{FPGo, FPRootOutliveAddrSrc})

		var toTmpEdge *escFlow
		for _, edge := range outedges {
			if edge != toHeapEdge {
				toTmpEdge = edge
			}
		}
		toNode := toTmpEdge.dst
		assert.Equal(t, 1, len(esg.EdgeFrom(toNode.id)))
		toHeapEdge, ok = esg.Edge(toNode.ID(), heap.ID()).(*escFlow)
		assert.NotNil(t, esg.Edge(toNode.ID(), heap.ID()))
		assert.True(t, ok)
		assert.Subset(t, toHeapEdge.whys, []string{FPRootEscape})

		wg := esg.findNode("wg @ main.go:7:6")
		assert.NotNil(t, wg)
		outedges = esg.EdgeFrom(wg.id)
		assert.Equal(t, 3, len(outedges))

		toHeapEdge, ok = esg.Edge(wg.ID(), heap.ID()).(*escFlow)
		assert.NotNil(t, toHeapEdge)
		assert.True(t, ok)
		assert.Subset(t, toHeapEdge.whys, []string{FPGo, FPRootOutliveAddrSrc})

		leftEdges := make([]*escFlow, 0, 2)
		for _, edge := range outedges {
			if edge != toHeapEdge {
				leftEdges = append(leftEdges, edge)
			}
		}
		toWgWaitEdgeId := 0
		if slices.Index(leftEdges[0].whys, FPGo) >= 0 {
			toWgWaitEdgeId = 1
		}
		wgWaitTmpNode := leftEdges[toWgWaitEdgeId].dst
		wgWaitTmp2HeapEdge, ok := esg.Edge(wgWaitTmpNode.ID(), heap.ID()).(*escFlow)
		assert.NotNil(t, wgWaitTmp2HeapEdge)
		assert.True(t, ok)
		assert.Subset(t, wgWaitTmp2HeapEdge.whys, []string{FPRootEscape})

		toIntTransferEdge := leftEdges[1-toWgWaitEdgeId]
		assert.NotNil(t, toIntTransferEdge)
		assert.Subset(t, toIntTransferEdge.whys, []string{FPGo})
		intTransferTmpNode := toIntTransferEdge.dst
		intTransferTmp2HeapEdge, ok := esg.Edge(intTransferTmpNode.ID(), heap.ID()).(*escFlow)
		assert.NotNil(t, intTransferTmp2HeapEdge)
		assert.True(t, ok)
		assert.Subset(t, intTransferTmp2HeapEdge.whys, []string{FPRootEscape})
	})

	scope(t, fixture.graph("main", 1, pSummarize), func(esg *escapeSubGraph) {
		a := esg.findNode("a @ main.go:6:2")
		assert.NotNil(t, a)
		assert.True(t, a.reason.contains(ERHeap|ERGo|ERClosure|ERHeap|ERParamLeak))

		wg := esg.findNode("wg @ main.go:7:6")
		assert.NotNil(t, wg)
		assert.True(t, wg.reason.contains(ERClosure|ERGo|ERHeap|ERParamLeak))
	})
}
