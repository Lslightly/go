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

type locInfo struct {
	outs map[hole]bool // destinations to which current location escapes to
}
type escapeLocations map[*location]*locInfo
type escapeSubGraph struct {
	elocs escapeLocations                     // escape related locations
	b     *batch                              // points to current batch analyzed
	gs    []graph.Graph[*location, *location] // whole program graphs
}

var _esg *escapeSubGraph = nil

func getEsg() *escapeSubGraph {
	return _esg
}

func createEsg() {
	_esg = &escapeSubGraph{
		elocs: make(escapeLocations),
		gs:    make([]graph.Graph[*location, *location], 0),
	}
}

func destroyEsg() {
	_esg = nil
}

func (esg *escapeSubGraph) getLocInfo(key *location) *locInfo {
	if info, ok := esg.elocs[key]; ok {
		return info
	}
	info := &locInfo{}
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

func (esg *escapeSubGraph) finishBatch() {
	g := graph.New(func(l *location) *location { return l })
	for src, info := range esg.elocs {
		if err := g.AddVertex(src, graph.VertexAttribute("explain", esg.b.explainLoc(src))); err != nil {
			continue
		}
		for hole := range info.outs {
			dst := hole.dst
			if err := g.AddVertex(dst, graph.VertexAttribute("explain", esg.b.explainLoc(dst))); err != nil {
				continue
			}
			g.AddEdge(src, dst, graph.EdgeAttribute("first-why", hole.notes.why))
		}
	}
	esg.gs = append(esg.gs, g)
}

func (esg *escapeSubGraph) dump() {
	for i, g := range esg.gs {
		os.MkdirAll("graphs", 0755)
		filename := fmt.Sprintf("graphs/escape_subgraph_%d.gv", i)
		svg := fmt.Sprintf("graphs/escape_subgraph_%d.svg", i)
		file, err := os.Create(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create subgraph %s failed\n", filename)
			continue
		}
		draw.DOT(g, file)
		exec.Command("dot", "-Tsvg", filename, "-o", svg).Run()
	}
}
