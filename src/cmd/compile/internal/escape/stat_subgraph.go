package escape

import (
	"cmd/compile/internal/ir"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	gn "gonum.org/v1/gonum/graph"
	encoding "gonum.org/v1/gonum/graph/encoding"
	gndot "gonum.org/v1/gonum/graph/encoding/dot"
)

const (
	FPDstEscAddrSrc      string = "dst = &src, dst(escape)"
	FPRootOutliveAddrSrc string = "root = &l, outlives(root, l)"
	FPParamLeak          string = "param l leak, outlives(root, l)"
	FPDefer              string = "defer"
	FPGo                 string = "go"
)

type escapeReason int64

const (
	ERHeap             escapeReason = 1 << iota // root is heap
	ERRootStack                                 // root in stack
	EROutliveRootStack                          // outlived by root in stack
	ERClousre                                   // root is closure
	ERReturn                                    // l is returned
	ERNumber           = iota
)

var reason2label map[escapeReason]string = map[escapeReason]string{
	ERHeap:             "h",
	ERRootStack:        "rs",
	EROutliveRootStack: "ors",
	ERClousre:          "clo",
	ERReturn:           "ret",
}

var reason2color map[escapeReason]string = map[escapeReason]string{
	ERHeap:             "slategrey",
	ERRootStack:        "lightgreen",
	EROutliveRootStack: "darkgreen",
	ERClousre:          "deepskyblue",
	ERReturn:           "gold",
}

func (er *escapeReason) add(r escapeReason) {
	*er |= r
}

func (er *escapeReason) del(r escapeReason) {
	*er &= ^r
}

func (er *escapeReason) contains(r escapeReason) bool {
	if *er&r == r {
		return true
	} else {
		return false
	}
}

var esgCnt int = 0

type escapeSubGraph struct {
	newid  int64 // new id to be generated
	loc2id map[*location]int64
	nodes  map[int64]*escLocNode
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
		newid:  0,
		loc2id: make(map[*location]int64),
		nodes:  make(map[int64]*escLocNode),
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
	id := esg.newid
	esg.newid++
	esg.loc2id[loc] = id
	node := &escLocNode{
		id:  id,
		loc: loc,
	}
	esg.nodes[id] = node
	esg.edges[id] = make(map[int64]*escFlow)
	return node
}

// Node returns the node with the given ID if it exists
// in the graph, and nil otherwise.
func (esg *escapeSubGraph) Node(id int64) gn.Node {
	if _, ok := esg.nodes[id]; !ok {
		return nil
	}
	return esg.nodes[id]
}

// Nodes returns all the nodes in the graph.
//
// Nodes must not return nil.
func (esg *escapeSubGraph) Nodes() gn.Nodes {
	nodes := make([]*escLocNode, 0, len(esg.nodes))
	for _, node := range esg.nodes {
		nodes = append(nodes, node)
	}
	return createIter(nodes)
}

// From returns all nodes that can be reached directly
// from the node with the given ID.
//
// From must not return nil.
func (esg *escapeSubGraph) From(id int64) gn.Nodes {
	flows := esg.EdgeFrom(id)
	nodes := make([]*escLocNode, 0, len(flows))
	for _, f := range flows {
		nodes = append(nodes, f.dst)
	}
	return createIter(nodes)
}

// EdgeFrom returns all edges from id
func (esg *escapeSubGraph) EdgeFrom(id int64) []*escFlow {
	if _, ok := esg.nodes[id]; !ok {
		return nil
	}
	for srcid, outs := range esg.edges {
		if srcid == id {
			edges := make([]*escFlow, 0, len(outs))
			for _, e := range outs {
				edges = append(edges, e)
			}
			return edges
		}
	}
	return nil
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
	flows := esg.EdgeTo(id)
	inNodes := make([]*escLocNode, 0, len(flows))
	for _, f := range flows {
		inNodes = append(inNodes, f.src)
	}
	return createIter(inNodes)
}

func (esg *escapeSubGraph) EdgeTo(id int64) []*escFlow {
	edges := make([]*escFlow, 0)
	for _, outs := range esg.edges {
		if e, ok := outs[id]; ok {
			edges = append(edges, e)
		}
	}
	return edges
}

type escLocNode struct {
	id     int64
	loc    *location
	dotid  string
	reason escapeReason
}

// returns a graph-unique integer ID.
func (n *escLocNode) ID() int64 {
	return n.id
}

// TODO add reason dump, test now
func (n *escLocNode) DOTID() string {
	if n.dotid == "" {
		n.dotid = getEsg().b.explainLocPos(n.loc)
	}
	return n.dotid
}

func (n *escLocNode) addReason(r escapeReason) {
	n.reason.add(r)
}

// add label and xlabel
//
// https://www.graphviz.org/docs/attrs/xlabel/
// https://www.graphviz.org/docs/attrs/color/
func (n *escLocNode) Attributes() []encoding.Attribute {
	resAttrs := make([]encoding.Attribute, 0)
	rlabels := make([]string, 0, ERNumber)
	for reason, label := range reason2label {
		if n.reason.contains(reason) {
			rlabels = append(rlabels, label)
		}
	}
	resAttrs = append(resAttrs, encoding.Attribute{
		Key:   "xlabel",
		Value: strings.Join(rlabels, ","),
	})
	colors := make([]string, 0, ERNumber)
	for reason, color := range reason2color {
		if n.reason.contains(reason) {
			colors = append(colors, color)
		}
	}
	if len(colors) != 0 {
		resAttrs = append(resAttrs,
			encoding.Attribute{
				Key:   "style",
				Value: "filled",
			},
			encoding.Attribute{
				Key:   "fillcolor",
				Value: strings.Join(colors, fmt.Sprintf(";%f:", 1.0/float32(len(colors)))),
			})
	}
	return resAttrs
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

// backPropogate returns back propogate reason to be added to src node
func (f *escFlow) backPropogate() escapeReason {
	dst := f.dst
	dstLoc := dst.loc
	r := dst.reason
	// TODO add more reason
	switch dst.loc.n.(type) {
	case *ir.ClosureExpr:
		r.add(ERClousre)
	}
	if dstLoc.isName(ir.PPARAMOUT) {
		r.add(ERReturn)
	}
	return r
}

type nodeRemover struct{}

// RemoveNode removes the node with the given ID
// from the graph, as well as any edges attached
// to it. If the node is not in the graph it is
// a no-op.
func (nr *nodeRemover) RemoveNode(rid int64) {
	esg := getEsg()
	if _, ok := esg.nodes[rid]; !ok {
		return
	}
	for loc, id := range esg.loc2id {
		if id == rid {
			delete(esg.loc2id, loc)
			break
		}
	}
	delete(esg.nodes, rid)
	for srcid, outs := range esg.edges {
		if srcid == rid { // out edge
			delete(esg.edges, srcid)
			break
		}
		for dstid := range outs { // in edge
			if dstid == rid {
				delete(esg.edges[srcid], dstid)
				break
			}
		}
	}
}

/*
why of the first note of hole must be FPDstEscAddrSrc/FPRootOutliveAddrSrc/FPParamLeak
*/
func (esg *escapeSubGraph) fastpath(src *location, hole hole) {
	dst := hole.dst
	srcN, dstN := esg.createOrGetNode(src), esg.createOrGetNode(dst)
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
	dumpRoot := path.Join("graphs", strconv.Itoa(esgCnt))
	os.MkdirAll(dumpRoot, 0755)
	esgCnt += 1
	esg.dump(dumpRoot, "0init")
	/*
		1. x -(d, y)-> heap
			remove heap(can be counted)
		2. back propogate from root to all source
			add node attribute(can be counted)
		3. for each source node, summarize all attributes(define priority)
	*/
	esg.removeHeap()
	esg.dump(dumpRoot, "1removeheap")
	esg.backPropogate()
	esg.dump(dumpRoot, "2backPropogate")
	// esg.summarize()
	// phase = "summarize"
	// esg.dump(esgCnt, phase)
}

func (esg *escapeSubGraph) dump(dumpRoot, phase string) {
	dotfile := path.Join(dumpRoot, phase+".gv")
	svg := path.Join(dumpRoot, phase+".svg")
	f, err := os.Create(dotfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create subgraph %s failed\n", dotfile)
		return
	}
	defer f.Close()
	bs, err := gndot.Marshal(esg, phase, "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal of %s failed\n", phase)
		return
	}
	n, err := f.Write(bs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write error %v\n", err)
	}
	fmt.Println(dotfile, "write bytes:", n)
	if err := exec.Command("dot", "-Tsvg", dotfile, fmt.Sprintf("-o%s", svg)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	for id, node := range esg.nodes {
		fmt.Printf("id %d: %s\n", id, node.DOTID())
		fmt.Printf("\treason %#b\n", node.reason)
	}
}

func (esg *escapeSubGraph) removeHeap() {
	nr := nodeRemover{}
	heapid := esg.loc2id[&esg.b.heapLoc]
	// add ERRootStackOutlive | ERRootStack for nodes which do have out edges, except heap node
	for id, node := range esg.nodes {
		if len(esg.edges[id]) == 0 && id != heapid {
			node.reason |= ERRootStack | EROutliveRootStack
		}
	}

	// add ERHeap for in nodes of heap node
	inNodesIter := esg.To(heapid)
	for inNodesIter.Next() { // add ERHeap for each node outlived by heap
		inNodesIter.Node().(*escLocNode).addReason(ERHeap)
	}
	nr.RemoveNode(heapid)
}

// backPropogate from nodes which do not have out edges
func (esg *escapeSubGraph) backPropogate() {
	wl := make([]*escLocNode, 0)
	// add nodes which do not have out edges
	for id, node := range esg.nodes {
		if len(esg.edges[id]) == 0 {
			wl = append(wl, node)
		}
	}
	for len(wl) != 0 {
		dst := wl[len(wl)-1] // first in last out
		wl = wl[:len(wl)-1]
		flows := esg.EdgeTo(dst.id)
		for _, f := range flows {
			r := f.backPropogate()
			f.src.addReason(r)
			wl = append(wl, f.src)
		}
	}
}

func (esg *escapeSubGraph) summarize() {
}
