/*
Escape Reason Statistics of SubGraph
*/
package escape

import (
	"cmd/compile/internal/base"
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

var statDebug bool = false

/*
enabled:
EAReasonStat=1 go build
go build -a -gcflags="-eareasonstat"

don't enable EAReason when building standard library

EAReasonStat=0 go build // disabled
*/
func isEAReasonStatEnabled() (res bool) {
	if base.Flag.Std {
		return false
	}
	v, ok := os.LookupEnv("EAReasonStat")
	res = res || (ok && v != "0")
	res = res || (base.Flag.EAReasonStat)
	return
}

/*
enabled:
EAReasonDebug=1 go build
go build -a -gcflags="-eareasondebug"

disabled:
EAReasonDebug=0 go build
*/
func isEAReasonDebug() (res bool) {
	v, ok := os.LookupEnv("EAReasonDebug")
	res = res || (ok && v != "0")
	res = res || base.Flag.EAReasonDebug
	return
}

func eaResultRoot() string {
	v, ok := os.LookupEnv("EAResultRoot")
	if ok {
		return v
	}
	return base.Flag.EAResultRoot
}

// FP: fast path
const (
	FPDstEscAddrSrc      string = "dst = &src, dst(escape)"
	FPRootOutliveAddrSrc string = "root = &l, outlives(root, l)"
	FPParamLeak          string = "param l leak, outlives(root, l)"
	FPDefer              string = "defer"
	FPGo                 string = "go"
	FPToIndex            string = "assign to S[Index]"
	FPToStar             string = "assign to *X"
	FPToDotPtr           string = "assign to (*X).Sel"
	FPToMapIndex         string = "assign to M[Index]"
)

// bitstring for escape reason
type escapeReason int64

const (
	ERHeap             escapeReason = 1 << iota // root is heap
	ERStar                                      // root is *x
	ERRootStack                                 // root in stack
	EROutliveRootStack                          // outlived by root in stack
	ERClosure                                   // root is closure
	ERGo                                        // root is go
	ERDefer                                     // root is defer
	ERReturn                                    // l is returned
	ERParamLeak                                 // param leak
	ERCallArgument                              // root is call argument
	ERMapIdx                                    // m[index]
	ERSliceIdx                                  // s[index]
	ERComm                                      // communication by send
	ERAppend                                    // append
	ERCopy                                      // copied slice
	ERMust                                      // must be heap allocated
	ERNumber           = iota
)

var reason2xlabel map[escapeReason]string = map[escapeReason]string{
	ERHeap:             "h",
	ERStar:             "*X",
	ERRootStack:        "rs",
	EROutliveRootStack: "ors",
	ERClosure:          "clo",
	ERGo:               "go",
	ERDefer:            "defer",
	ERReturn:           "ret",
	ERParamLeak:        "param",
	ERCallArgument:     "arg",
	ERMapIdx:           "mapidx",
	ERSliceIdx:         "sliceidx",
	ERComm:             "send",
	ERAppend:           "append",
	ERCopy:             "copy",
	ERMust:             "must",
}

var reason2fillcolor map[escapeReason]string = map[escapeReason]string{
	ERHeap:             "slategrey",
	ERStar:             "pink",
	ERRootStack:        "lightgreen",
	EROutliveRootStack: "darkgreen",
	ERClosure:          "darkgreen",
	ERDefer:            "darkgreen",
	ERGo:               "deepskyblue",
	ERComm:             "deepskyblue",
	ERReturn:           "gold",
	ERParamLeak:        "gold",
	ERCallArgument:     "gold",
	ERMapIdx:           "red",
	ERSliceIdx:         "red",
	ERAppend:           "red",
	ERCopy:             "red",
	ERMust:             "red",
}

var why2reason map[string]escapeReason = map[string]escapeReason{
	FPParamLeak:             ERParamLeak,
	FPDefer:                 ERDefer,
	FPGo:                    ERGo,
	FPToIndex:               ERSliceIdx,
	FPToStar:                ERStar,
	FPToDotPtr:              ERStar,
	FPToMapIndex:            ERMapIdx,
	"key of map put":        ERMapIdx,
	"send":                  ERComm,
	"return":                ERReturn,
	"appendee slice":        ERAppend,
	"appended slice...":     ERAppend,
	"copied slice":          ERCopy,
	"too large for stack":   ERMust,
	"too aligned for stack": ERMust,
	"non-constant size":     ERMust,
	"reference":             ERClosure,
	"call parameter":        ERCallArgument,
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

func (er escapeReason) towhys() []string {
	whys := make([]string, 0)
	for why, reason := range why2reason {
		if er.contains(reason) {
			whys = append(whys, why)
		}
	}
	return whys
}

var esgCnt int = 0

type escapeSubGraph struct {
	newid  int64               // new id to be generated
	loc2id map[*location]int64 // location -> srcid
	nodes  map[int64]*escLocNode
	edges  map[int64]map[int64]*escFlow // srcid -> dstid -> flow
	b      *batch                       // points to current batch analyzed
}

var _defaultesg = escapeSubGraph{}
var _esg *escapeSubGraph = &_defaultesg

func getEsg() *escapeSubGraph {
	return _esg
}

func createEsg(b *batch) {
	_esg = &escapeSubGraph{
		newid:  1, // start from 1 to avoid confusing default value 0 of loc2id[not exist location] with valid location nodeid
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

// escape subgraph node
type escLocNode struct {
	id     int64 // start from 1
	loc    *location
	label  string // cache field for Label()
	reason escapeReason
}

// returns a graph-unique integer ID.
func (n *escLocNode) ID() int64 {
	return n.id
}

// TODO add reason dump, test now
// implement dot.Node
func (n *escLocNode) Label() string {
	if n.label == "" {
		n.label = getEsg().b.explainLocPos(n.loc)
	}
	return n.label
}

func (n *escLocNode) addReason(r escapeReason) (changed bool) {
	oldReason := n.reason
	n.reason.add(r)
	n.reason.del(ERRootStack) // ERRootStack should be set manually
	if statDebug {
		fmt.Fprintf(os.Stderr, "Update reason(id: %d): %#b -> %#b\n", n.id, oldReason, n.reason)
	}
	return oldReason != n.reason
}

// add label and xlabel
//
// https://www.graphviz.org/docs/attrs/
func (n *escLocNode) Attributes() []encoding.Attribute {
	resAttrs := make([]encoding.Attribute, 0)
	resAttrs = append(resAttrs, encoding.Attribute{
		Key:   "label",
		Value: n.Label(),
	})
	rlabels := make([]string, 0, ERNumber)
	for reason, label := range reason2xlabel {
		if n.reason.contains(reason) {
			rlabels = append(rlabels, label)
		}
	}
	resAttrs = append(resAttrs, encoding.Attribute{
		Key:   "xlabel",
		Value: strings.Join(rlabels, ","),
	})
	colors := make([]string, 0, ERNumber)
	for reason, color := range reason2fillcolor {
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

// call Next first and then Node to get the node from iterator
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

// escape subgraph edge
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
	r := dst.reason
	// TODO add more reason
	for note := f.hole.notes; note != nil; note = note.next {
		if reason, ok := why2reason[note.why]; ok {
			r.add(reason)
		}
		if statDebug {
			fmt.Fprintln(os.Stderr, "where", note.where)
			fmt.Fprintln(os.Stderr, "why", note.why)
		}
	}
	switch dst.loc.n.(type) {
	case *ir.ClosureExpr:
		if dst.loc.n.(*ir.ClosureExpr).IsGoWrap {
			r.add(ERGo)
		}
		r.add(ERClosure)
	}
	if f.src.loc.isName(ir.PPARAM) {
		r.add(ERParamLeak)
	}
	if dst.loc.isName(ir.PPARAMOUT) {
		r.add(ERReturn)
	}
	return r
}

func (f *escFlow) Attributes() []encoding.Attribute {
	resAttrs := make([]encoding.Attribute, 0)
	whys := make([]string, 0)
	for note := f.hole.notes; note != nil; note = note.next {
		whys = append(whys, note.why)
	}
	resAttrs = append(resAttrs, encoding.Attribute{
		Key:   "label",
		Value: strconv.Itoa(f.hole.derefs) + "\nwhy:\n" + strings.Join(whys, "\n"),
	})
	return resAttrs
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
	if statDebug {
		fmt.Fprintln(os.Stderr, "remove node", rid, esg.nodes[rid].Label())
	}
	delete(esg.nodes, rid)
	for srcid := range esg.edges {
		delete(esg.edges[srcid], rid)
	}
	delete(esg.edges, rid) // delete out edges from rid
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

func esgLog(msg string) {
	if _, err := os.Stat(esgLogPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s does not exist\n", esgLogPath)
		return
	}
	f, err := os.OpenFile(esgLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o664)
	if err != nil {
		fmt.Fprintf(os.Stderr, "append %s failed\n", esgLogPath)
	}
	defer f.Close()
	fmt.Fprintln(f, msg)
}

func (esg *escapeSubGraph) finishBatch() {
	if len(esg.nodes) == 0 {
		return
	}
	graphBatchRoot := path.Join(graphRoot, strconv.Itoa(esgCnt))
	csvBatchRoot := path.Join(csvRoot, strconv.Itoa(esgCnt))
	if err := os.MkdirAll(graphBatchRoot, 0o755); err != nil {
		esgLog(fmt.Sprintf("FAIL: graph batch root %s", graphBatchRoot))
	} else {
		esgLog(fmt.Sprintf("SUCC: graph batch root %s", graphBatchRoot))
	}
	if err := os.MkdirAll(csvBatchRoot, 0o755); err != nil {
		esgLog(fmt.Sprintf("FAIL: csv batch root %s", csvBatchRoot))
	} else {
		esgLog(fmt.Sprintf("SUCC: csv batch root %s", csvBatchRoot))
	}
	esgCnt += 1
	esg.dump(graphBatchRoot, phase2name[0])
	/*
		1. x -(d, y)-> heap
			remove heap(can be counted)
		2. back propogate from root to all source
			add node attribute(can be counted)
		3. for each source node, summarize all attributes(define priority)
	*/
	esg.removeHeap()
	esg.dump(graphBatchRoot, phase2name[1])
	esg.backPropogate()
	esg.dump(graphBatchRoot, phase2name[2])
	esg.summarize(csvBatchRoot)
	esg.dump(graphBatchRoot, phase2name[3])
}

var graphRoot, csvRoot, esgLogPath string

func initEAResultRoot() {
	graphRoot = path.Join(eaResultRoot(), "graphs", base.Ctxt.Pkgpath)
	csvRoot = path.Join(eaResultRoot(), "csv", base.Ctxt.Pkgpath)
	esgLogsRoot := path.Join(eaResultRoot(), "logs", base.Ctxt.Pkgpath)
	for _, root := range []string{graphRoot, csvRoot, esgLogsRoot} {
		os.MkdirAll(root, 0o755)
	}
	esgLogPath = path.Join(esgLogsRoot, "esg.log")
	f, err := os.Create(esgLogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create %s failed", esgLogPath)
	}
	defer f.Close()
}

var phase2name [4]string = [4]string{
	0: "0init",
	1: "1removeheap",
	2: "2backPropogate",
	3: "summarize",
}

func (esg *escapeSubGraph) dump(dumpRoot, phase string) {
	dotfile := path.Join(dumpRoot, phase+".gv")
	svg := path.Join(dumpRoot, phase+".svg")

	// dump dot graph
	bs, err := gndot.Marshal(esg, phase, "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal of %s failed\n", phase)
		return
	}
	n, err := writeToFile(dotfile, string(bs))
	if err != nil {
		fmt.Fprintf(os.Stderr, "write error %v\n", err)
	}
	if statDebug {
		fmt.Fprintln(os.Stderr, dotfile, "write bytes:", n)
	}

	// create svg
	if err := exec.Command("dot", "-Tsvg", dotfile, fmt.Sprintf("-o%s", svg)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	if statDebug {
		// print node info
		for id, node := range esg.nodes {
			fmt.Fprintf(os.Stderr, "id %d: %s\n", id, node.Label())
			fmt.Fprintf(os.Stderr, "\treason %#b\n", node.reason)
		}
	}
}

/*
removeHeap first mark all nodes which does not have out edges
as ERRootStack. Then it propogate ERHeap to each node outlived
by heap. Finally it removes the heap node from the subgraph.
*/
func (esg *escapeSubGraph) removeHeap() {
	nr := nodeRemover{}
	heapid, ok := esg.loc2id[&esg.b.heapLoc] // return default id 0 if heap does not exist
	// add ERRootStackOutlive | ERRootStack for nodes which do have out edges, except heap node
	for id, node := range esg.nodes {
		if len(esg.edges[id]) == 0 && id != heapid {
			if statDebug {
				fmt.Fprintln(os.Stderr, "find root stack node id:", id, "label:", node.Label())
			}
			node.reason |= ERRootStack | EROutliveRootStack
		}
	}

	if !ok { // heap node does not exist
		if statDebug {
			fmt.Fprintln(os.Stderr, "heap does not exist")
		}
		return
	}

	// add ERHeap for in nodes of heap node
	flows := esg.EdgeTo(heapid)
	for _, f := range flows {
		r := f.backPropogate()
		f.src.addReason(r)
		f.src.addReason(ERHeap)
	}
	nr.RemoveNode(heapid)
}

/*
backPropogate starting from nodes which do not have out edges.
It traverses in topological order of edge-reversed graph
*/
func (esg *escapeSubGraph) backPropogate() {
	wl := make([]*escLocNode, 0)
	// add nodes which do not have out edges
	for id, node := range esg.nodes {
		if len(esg.edges[id]) == 0 {
			wl = append(wl, node)
		}
	}
	for id, node := range esg.nodes { // then add nodes which have out edges
		if len(esg.edges[id]) != 0 {
			wl = append(wl, node)
		}
	}
	for len(wl) != 0 {
		dst := wl[len(wl)-1] // first in last out
		wl = wl[:len(wl)-1]
		flows := esg.EdgeTo(dst.id)
		for _, f := range flows {
			r := f.backPropogate()
			if f.src.addReason(r) {
				wl = append(wl, f.src)
			}
		}
	}
}

func (esg *escapeSubGraph) summarize(csvBatchRoot string) {
	for _, node := range esg.nodes {
		if node.reason.contains(ERRootStack | EROutliveRootStack) {
			node.reason.del(EROutliveRootStack)
		}
	}

	nrLines := make([]string, 0, len(esg.nodes))
	for _, node := range esg.nodes {
		locPos := "<nil>"
		if node.loc.n != nil {
			locPos = base.FmtPos(node.loc.n.Pos())
		}
		nrLines = append(nrLines, fmt.Sprintf("%d,%d,%s", node.id, node.reason, locPos))
	}

	content := fmt.Sprintf(`nodeid,reason,loc
%s
`, strings.Join(nrLines, "\n"))
	writeToFile(path.Join(csvBatchRoot, "all.csv"), content)
}

func writeToFile(fpath string, content string) (int, error) {
	f, err := os.Create(fpath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create", fpath, "failed")
		return 0, err
	}
	defer f.Close()
	return fmt.Fprint(f, content)
}

func (esg *escapeSubGraph) encodeLocLeakReason(loc *location, esc *leaks) {
	id, ok := esg.loc2id[loc]
	if !ok {
		return
	}
	node := esg.nodes[id]
	esc.encodeReason(node.reason)
}
