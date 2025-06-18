package escape

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

const (
	FPDstEscAddrSrc      string = "dst = &src, dst(escape)"
	FPRootOutliveAddrSrc string = "root = &l, outlives(root, l)"
	FPParamLeak          string = "param l leak, outlives(root, l)"
	FPDefer              string = "defer"
	FPGo                 string = "go"
)

var esgCnt int = 0

type locInfo struct {
	outs map[hole]bool // destinations to which current location escapes to
}
type escapeLocations map[*location]*locInfo
type escapeSubGraph struct {
	elocs escapeLocations // escape related locations
	b     *batch          // points to current batch analyzed
}

var _defaultesg = escapeSubGraph{}
var _esg *escapeSubGraph = &_defaultesg

func getEsg() *escapeSubGraph {
	return _esg
}

func createEsg(b *batch) {
	_esg = &escapeSubGraph{
		elocs: make(escapeLocations),
		b:     b,
	}
}

func destroyEsg() {
	_esg = nil
}

func (esg *escapeSubGraph) getLocInfo(key *location) *locInfo {
	if info, ok := esg.elocs[key]; ok {
		return info
	}
	info := &locInfo{
		outs: make(map[hole]bool),
	}
	esg.elocs[key] = info
	return info
}

/*
why of the first note of hole must be FPDstEscAddrSrc/FPRootOutliveAddrSrc/FPParamLeak
*/
func (esg *escapeSubGraph) fastpath(src *location, hole hole) {
	srcLocInfo := esg.getLocInfo(src)
	srcLocInfo.outs[hole] = true
}

type locGraph graph.Graph[string, *location]

func (esg *escapeSubGraph) finishBatch() {
	g := esg.createGraph()
	phase := "init"
	esg.dump(g, esgCnt, phase)
	esgCnt += 1
	/*
		1. x -(d, y)-> heap
			remove heap(can be counted)
		2. back propogate from root to all source
			add node attribute(can be counted)
		3. for each source node, summarize all attributes(define priority)
	*/
	g = esg.removeHeap(g)
	phase = "removeheap"
	esg.dump(g, esgCnt, phase)
	g = esg.backPropogate(g)
	phase = "backPropogate"
	esg.dump(g, esgCnt, phase)
	g = esg.summarize(g)
	phase = "summarize"
	esg.dump(g, esgCnt, phase)
}

func (esg *escapeSubGraph) createGraph() locGraph {
	hash := func(l *location) string { return esg.b.explainLocPos(l) }
	g := graph.New(hash, graph.Directed())
	for src, info := range esg.elocs {
		g.AddVertex(src)
		for hole := range info.outs {
			dst := hole.dst
			g.AddVertex(dst)
			g.AddEdge(hash(src), hash(dst), graph.EdgeAttribute("label", hole.notes.why))
		}
	}
	return g
}

func (esg *escapeSubGraph) dump(g locGraph, esgId int, phase string) {
	os.MkdirAll("graphs", 0755)
	filename := fmt.Sprintf("graphs/escape_subgraph_%d_%s.gv", esgId, phase)
	svg := fmt.Sprintf("graphs/escape_subgraph_%d_%s.svg", esgId, phase)
	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create subgraph %s failed\n", filename)
		return
	}
	draw.DOT(g, file)
	if err := exec.Command("dot", "-Tsvg", filename, fmt.Sprintf("-o%s", svg)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (esg *escapeSubGraph) removeHeap(g locGraph) locGraph {
	return g
}

func (esg *escapeSubGraph) backPropogate(g locGraph) locGraph {
	return g
}

func (esg *escapeSubGraph) summarize(g locGraph) locGraph {
	return g
}
