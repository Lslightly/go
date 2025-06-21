package escape

import (
	"fmt"
	"os"
	"os/exec"

	gn "gonum.org/v1/gonum/graph"
	gndot "gonum.org/v1/gonum/graph/encoding/dot"
)

const (
	FPDstEscAddrSrc      string = "dst = &src, dst(escape)"
	FPRootOutliveAddrSrc string = "root = &l, outlives(root, l)"
	FPParamLeak          string = "param l leak, outlives(root, l)"
	FPDefer              string = "defer"
	FPGo                 string = "go"
)

var esgCnt int = 0

type escapeSubGraph struct {
	loc2id map[*location]int64
	nodes  []*escLocNode
	edges  map[int64]map[int64]*escFlow
	b      *batch // points to current batch analyzed
}

var _defaultesg = escapeSubGraph{}
var _esg *escapeSubGraph = &_defaultesg

func getEsg() *escapeSubGraph {
	return _esg
}

func createEsg(b *batch) {
	_esg = &escapeSubGraph{
		loc2id: make(map[*location]int64),
		nodes:  make([]*escLocNode, 0),
		edges:  make(map[int64]map[int64]*escFlow),
		b:      b,
	}
}

func destroyEsg() {
	_esg = nil
}

func (esg *escapeSubGraph) createOrGetNode(loc *location) *escLocNode {
	if id, ok := esg.loc2id[loc]; ok {
		return esg.nodes[id]
	}
	id := int64(len(esg.nodes))
	esg.loc2id[loc] = id
	node := &escLocNode{
		id:      id,
		loc:     loc,
		inNodes: make(map[*escLocNode]bool),
	}
	esg.nodes = append(esg.nodes, node)
	esg.edges[id] = make(map[int64]*escFlow)
	return node
}

// Node returns the node with the given ID if it exists
// in the graph, and nil otherwise.
func (esg *escapeSubGraph) Node(id int64) gn.Node {
	if id > int64(len(esg.nodes)) {
		return nil
	}
	return esg.nodes[id]
}

// Nodes returns all the nodes in the graph.
//
// Nodes must not return nil.
func (esg *escapeSubGraph) Nodes() gn.Nodes {
	return createIter(esg.nodes)
}

// From returns all nodes that can be reached directly
// from the node with the given ID.
//
// From must not return nil.
func (esg *escapeSubGraph) From(id int64) gn.Nodes {
	if id > int64(len(esg.nodes)) {
		return createIter(make([]*escLocNode, 0))
	}
	outs := esg.edges[id]
	nodes := make([]*escLocNode, 0, len(outs))
	for _, flow := range outs {
		nodes = append(nodes, flow.dst)
	}
	return createIter(nodes)
}

// HasEdgeBetween returns whether an edge exists between
// nodes with IDs xid and yid without considering direction.
func (esg *escapeSubGraph) HasEdgeBetween(xid, yid int64) bool {
	return esg.Edge(xid, yid) != nil
}

// Edge returns the edge from u to v, with IDs uid and vid,
// if such an edge exists and nil otherwise. The node v
// must be directly reachable from u as defined by the
// From method.
func (esg *escapeSubGraph) Edge(uid, vid int64) gn.Edge {
	outs := esg.edges[uid]
	edge, exist := outs[vid]
	if exist {
		return edge
	} else {
		return nil
	}
}

// HasEdgeFromTo returns whether an edge exists
// in the graph from u to v with IDs uid and vid.
func (esg *escapeSubGraph) HasEdgeFromTo(uid, vid int64) bool {
	return esg.HasEdgeBetween(uid, vid)
}

// To returns all nodes that can reach directly
// to the node with the given ID.
//
// To must not return nil.
func (esg *escapeSubGraph) To(id int64) gn.Nodes {
	inNodes := make([]*escLocNode, 0, len(esg.nodes[id].inNodes))
	for node := range esg.nodes[id].inNodes {
		inNodes = append(inNodes, node)
	}
	return createIter(inNodes)
}

type escLocNode struct {
	id      int64
	loc     *location
	inNodes map[*escLocNode]bool
}

// returns a graph-unique integer ID.
func (n *escLocNode) ID() int64 {
	return n.id
}

type escLocNodeIter struct {
	nodes []*escLocNode
	cur   int
}

func createIter(nodes []*escLocNode) *escLocNodeIter {
	return &escLocNodeIter{
		nodes: nodes,
		cur:   -1,
	}
}

// Next advances the iterator and returns whether
// the next call to the item method will return a
// non-nil item.
//
// Next should be called prior to any call to the
// iterator's item retrieval method after the
// iterator has been obtained or reset.
func (iter *escLocNodeIter) Next() bool {
	if iter.cur+1 < len(iter.nodes) {
		iter.cur++
		return true
	} else {
		iter.cur = len(iter.nodes) - 1
		return false
	}
}

// Len returns the number of items remaining in the
// iterator.
func (iter *escLocNodeIter) Len() int {
	return len(iter.nodes) - iter.cur
}

// Reset returns the iterator to its start position.
func (iter *escLocNodeIter) Reset() {
	iter.cur = -1
}

// Node returns the current Node from the iterator.
func (iter *escLocNodeIter) Node() gn.Node {
	if iter.cur < 0 || iter.cur >= len(iter.nodes) {
		return nil
	}
	return iter.nodes[iter.cur]
}

type escFlow struct {
	src  *escLocNode
	dst  *escLocNode
	hole hole
}

// From returns the from node of the edge.
func (f *escFlow) From() gn.Node {
	return f.src
}

// To returns the to node of the edge.
func (f *escFlow) To() gn.Node {
	return f.dst
}

/*
reverse is invalid
*/
func (f *escFlow) ReversedEdge() gn.Edge {
	return f
}

/*
why of the first note of hole must be FPDstEscAddrSrc/FPRootOutliveAddrSrc/FPParamLeak
*/
func (esg *escapeSubGraph) fastpath(src *location, hole hole) {
	dst := hole.dst
	srcN, dstN := esg.createOrGetNode(src), esg.createOrGetNode(dst)
	dstN.inNodes[srcN] = true
	esg.edges[srcN.id][dstN.id] = &escFlow{
		src:  srcN,
		dst:  dstN,
		hole: hole,
	}
}

func (esg *escapeSubGraph) finishBatch() {
	if len(esg.nodes) == 0 {
		return
	}
	phase := "init"
	esg.dump(esgCnt, phase)
	esgCnt += 1
	/*
		1. x -(d, y)-> heap
			remove heap(can be counted)
		2. back propogate from root to all source
			add node attribute(can be counted)
		3. for each source node, summarize all attributes(define priority)
	*/
	// esg.removeHeap()
	// phase = "removeheap"
	// esg.dump(esgCnt, phase)
	// esg.backPropogate()
	// phase = "backPropogate"
	// esg.dump(esgCnt, phase)
	// esg.summarize()
	// phase = "summarize"
	// esg.dump(esgCnt, phase)
}

func (esg *escapeSubGraph) dump(esgId int, phase string) {
	os.MkdirAll("graphs", 0755)
	esgName := fmt.Sprintf("%d_%s", esgId, phase)
	filename := fmt.Sprintf("graphs/escape_subgraph_%s.gv", esgName)
	svg := fmt.Sprintf("graphs/escape_subgraph_%s.svg", esgName)
	f, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create subgraph %s failed\n", filename)
		return
	}
	defer f.Close()
	fmt.Println(len(esg.nodes))
	bs, err := gndot.Marshal(esg, esgName, "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal of %s failed\n", esgName)
		return
	}
	n, err := f.Write(bs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write error %v\n", err)
	}
	fmt.Println(filename, n)
	if err := exec.Command("dot", "-Tsvg", filename, fmt.Sprintf("-o%s", svg)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (esg *escapeSubGraph) removeHeap() {
}

func (esg *escapeSubGraph) backPropogate() {
}

func (esg *escapeSubGraph) summarize() {
}
