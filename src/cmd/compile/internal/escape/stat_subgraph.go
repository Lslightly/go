/*
Escape Reason Statistics of SubGraph
*/
package escape

import (
	"cmd/compile/internal/base"
	"cmd/compile/internal/ir"
	"context"
	"flag"
	"fmt"
	"log"
	"math/bits"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

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
ESGDumpEnabled=1 go build

disabled:
go build
*/
func isESGDumpEnabled() (res bool) {
	if base.Flag.Std {
		return false
	}
	v, ok := os.LookupEnv("ESGDumpEnabled")
	res = res || (ok && v != "0")
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
	FPToMapIndex         string = "assign to M[Index]"
	FPGlobal             string = "global"
)

// bitstring for escape reason
type escapeReason int64

const (
	ERHeap             escapeReason = 1 << iota // root is heap
	ERStar                                      // root is *x, here we don't add (*x).sel because now we don't care about field-sensitivity
	ERRootStack                                 // root in stack
	EROutliveRootStack                          // outlived by root in stack

	/*
		TODO: There are cases that are ERGo but not ERClosure
	*/

	ERClosure   // root is closure
	ERGo        // root is go
	ERDefer     // root is defer
	ERReturn    // l is returned
	ERParamLeak // param leak. arguments will also be marked as param leak if the param is param leak. Note ParamLeak does not necessarily mean parameter itself leaks to heap. Derefs of param can escapes to heap, so parameter should not be marked as escape.
	ERMapIdx    // m[index]
	ERSliceIdx  // s[index]
	ERComm      // communication by send
	ERAppend    // append
	ERCopy      // copied slice
	ERMust      // must be heap allocated
	ERGlobal    // flows to global variable
	ERNumber    = iota
)

var reason2str map[escapeReason]string = map[escapeReason]string{
	ERHeap:             "ERHeap",
	ERStar:             "ERStar",
	ERRootStack:        "ERRootStack",
	EROutliveRootStack: "EROutliveRootStack",
	ERClosure:          "ERClosure",
	ERGo:               "ERGo",
	ERDefer:            "ERDefer",
	ERReturn:           "ERReturn",
	ERParamLeak:        "ERParamLeak",
	ERMapIdx:           "ERMapIdx",
	ERSliceIdx:         "ERSliceIdx",
	ERComm:             "ERComm",
	ERAppend:           "ERAppend",
	ERCopy:             "ERCopy",
	ERMust:             "ERMust",
	ERGlobal:           "ERGlobal",
}

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
	ERMapIdx:           "mapidx",
	ERSliceIdx:         "sliceidx",
	ERComm:             "send",
	ERAppend:           "append",
	ERCopy:             "copy",
	ERMust:             "must",
	ERGlobal:           "global",
}

func xlabel2reason(xlabel string) (escapeReason, bool) {
	for k, v := range reason2xlabel {
		if v == xlabel {
			return k, true
		}
	}
	return ERNumber, false // not found
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
	ERMapIdx:           "red",
	ERSliceIdx:         "red",
	ERAppend:           "red",
	ERCopy:             "red",
	ERMust:             "red",
	ERGlobal:           "red",
}

var why2reason map[string]escapeReason = map[string]escapeReason{
	FPParamLeak:             ERParamLeak,
	FPDefer:                 ERDefer,
	FPGo:                    ERGo,
	FPToIndex:               ERSliceIdx,
	FPToStar:                ERStar,
	FPToMapIndex:            ERMapIdx,
	FPDstEscAddrSrc:         ERHeap,
	FPRootOutliveAddrSrc:    ERHeap, // root = &l, outlives(root, l)  should also be considered as ERHeap when backPropogate
	FPGlobal:                ERGlobal,
	"key of map put":        ERMapIdx,
	"send":                  ERComm,
	"return":                ERReturn,
	"appendee slice":        ERAppend,
	"appended slice...":     ERAppend,
	"copied slice":          ERCopy,
	"too large for stack":   ERMust,
	"too aligned for stack": ERMust,
	"non-constant size":     ERMust,
	"captured by a closure": ERClosure,
}

func createEscapeReason(low, high uint8) escapeReason {
	return escapeReason(uint16(low) | uint16(high)<<8)
}

func (er escapeReason) add(r escapeReason) escapeReason {
	er |= r
	return er
}

func (er escapeReason) del(r escapeReason) escapeReason {
	er &= ^r
	return er
}

// containAll returns true if er contains all reasons in r
func (er escapeReason) containAll(r escapeReason) bool {
	if er&r == r {
		return true
	} else {
		return false
	}
}

// containAnyOf returns true if er contains any one reason in r
func (er escapeReason) containAnyOf(r escapeReason) bool {
	return er&r != 0
}

func (er escapeReason) count() int {
	return bits.OnesCount64(uint64(er))
}

func (er escapeReason) towhys() []string {
	whys := make([]string, 0)
	// FIXME: there are many whys mapped to the same reason. For example, ERHeap, ERAppend and ERMust
	for why, reason := range why2reason {
		if er.containAll(reason) {
			whys = append(whys, why)
		}
	}
	return whys
}

func (er escapeReason) empty() bool {
	return er == 0
}

func (er escapeReason) String() string {
	strs := make([]string, 0, ERNumber)
	for i := range ERNumber {
		singleReason := escapeReason(1 << i)
		if er.containAll(singleReason) {
			strs = append(strs, reason2str[singleReason])
		}
	}
	return strings.Join(strs, ", ")
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
var _esgDumpWaitGrp sync.WaitGroup

func getEsg() *escapeSubGraph {
	return _esg
}

func waitDump() {
	_esgDumpWaitGrp.Wait()
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

// NewNode returns a new Node with a unique
// arbitrary ID.
// To implemnt NodeAdder interface
func (esg *escapeSubGraph) NewNode() gn.Node {
	id := esg.newid
	esg.newid++
	node := &escLocNode{
		id:  id,
		loc: nil,
	}
	return node
}

// AddNode adds a node to the graph. AddNode panics if
// the added node ID matches an existing node ID.
// To implemnt NodeAdder interface
func (esg *escapeSubGraph) AddNode(n gn.Node) {
	nid := n.ID()
	if _, ok := esg.nodes[nid]; ok {
		panic(fmt.Sprintf("node %d already exists in escapeSubGraph\n", nid))
	}
	esg.nodes[nid] = n.(*escLocNode)
	esg.edges[nid] = make(map[int64]*escFlow)
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

// NewEdge returns a new Edge from the source to the destination node.
// To implement EdgeAdder interface
func (esg *escapeSubGraph) NewEdge(from, to gn.Node) gn.Edge {
	return &escFlow{
		src:   from.(*escLocNode),
		dst:   to.(*escLocNode),
		holes: make([]hole, 0),
	}
}

// SetEdge adds an edge from one node to another.
// If the graph supports node addition the nodes
// will be added if they do not exist, otherwise
// SetEdge will panic.
// The behavior of an EdgeAdder when the IDs
// returned by e.From() and e.To() are equal is
// implementation-dependent.
// Whether e, e.From() and e.To() are stored
// within the graph is implementation dependent.
// To implement EdgeAdder interface
func (esg *escapeSubGraph) SetEdge(e gn.Edge) {
	from, to := e.From(), e.To()
	for _, n := range []gn.Node{from, to} {
		if v := esg.Node(n.ID()); v == nil {
			esg.AddNode(n)
		}
	}
	esg.edges[from.ID()][to.ID()] = e.(*escFlow)
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

func (esg *escapeSubGraph) Flow(u, v *escLocNode) *escFlow {
	flow, _ := esg.Edge(u.id, v.id).(*escFlow)
	return flow
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

func (n *escLocNode) IsTmp() bool {
	return n.loc == nil || n.loc.n == nil
}

func (n *escLocNode) IsParsedTmp() bool {
	return strings.Contains(n.label, "{temp ")
}

func (n *escLocNode) String() string {
	return fmt.Sprintf("node %d\n  label: %s\n  reason: %s\n", n.id, n.Label(), n.reasonsStr())
}

func (n *escLocNode) addReason(r escapeReason) (changed bool) {
	oldReason := n.reason
	n.reason = n.reason.add(r).del(ERRootStack) // ERRootStack should be set manually
	if statDebug {
		fmt.Fprintf(os.Stderr, "Update reason(id: %d): %#b -> %#b\n", n.id, oldReason, n.reason)
	}
	return oldReason != n.reason
}

func (n *escLocNode) reasonsStr() string {
	rlabels := make([]string, 0, ERNumber)
	for reason, label := range reason2xlabel {
		if n.reason.containAll(reason) {
			rlabels = append(rlabels, label)
		}
	}
	return strings.Join(rlabels, ",")
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
	resAttrs = append(resAttrs, encoding.Attribute{
		Key:   "xlabel",
		Value: n.reasonsStr(),
	})
	colors := make([]string, 0, ERNumber)
	for reason, color := range reason2fillcolor {
		if n.reason.containAll(reason) {
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

// Used by test
func (n *escLocNode) SetAttribute(attr encoding.Attribute) error {
	switch attr.Key {
	case "label":
		n.label = attr.Value
	case "xlabel":
		if len(attr.Value) == 0 {
			break
		}
		for rlabel := range strings.SplitSeq(attr.Value, ",") {
			r, ok := xlabel2reason(rlabel)
			if !ok {
				return fmt.Errorf("%s is not reason", rlabel)
			}
			n.reason |= r // always OR here because addReason will delete some reasons
		}
	}
	return nil
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
	src   *escLocNode
	dst   *escLocNode
	holes []hole
	// cached whys of hole, also used for test.
	// Notice that the field contains why which is not in why2reason.
	// But when backPropogate, only why in why2reason will be back propogated.
	whys []string
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
	for _, why := range f.whys {
		if reason, ok := why2reason[why]; ok {
			r = r.add(reason)
			if statDebug {
				getLogger().Println("why", why)
			}
		}
	}
	switch dst.loc.n.(type) {
	case *ir.ClosureExpr:
		if dst.loc.n.(*ir.ClosureExpr).IsGoWrap {
			r = r.add(ERGo)
		}
	}
	/*
		dst <- f - src
		src is Param
		dst Heap, f does not contain FPDstEscAddrSrc or FPRootOutliveAddrSrc. ERHeap should not appear in src.
	*/
	if dst.reason.containAnyOf(ERHeap) {
		if f.src.loc != nil && f.src.loc.isName(ir.PPARAM) {
			movedToHeapSet := map[string]bool{
				FPDstEscAddrSrc:      true,
				FPRootOutliveAddrSrc: true,
			}
			exist := false
			for _, why := range f.whys {
				if movedToHeapSet[why] {
					exist = true
				}
			}
			if !exist { // no movedToHeap why
				r = r.del(ERHeap)
			}
		}
	}
	return r
}

func (f *escFlow) Attributes() []encoding.Attribute {
	resAttrs := make([]encoding.Attribute, 0)
	resAttrs = append(resAttrs, encoding.Attribute{
		Key:   "label",
		Value: strings.Join(f.whys, "\n"),
	})
	resAttrs = append(resAttrs, encoding.Attribute{
		Key: "xlabel",
		Value: func() string {
			derefStrs := make([]string, 0, len(f.holes))
			for _, hole := range f.holes {
				derefStrs = append(derefStrs, strconv.Itoa(hole.derefs))
			}
			return strings.Join(derefStrs, ", ")
		}(),
	})
	return resAttrs
}

func (f *escFlow) SetAttribute(attr encoding.Attribute) error {
	switch attr.Key {
	case "label":
		f.whys = append(f.whys, strings.Split(attr.Value, "\n")...)
	}
	return nil
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
		getLogger().Println("remove node", rid, esg.nodes[rid].Label())
	}
	delete(esg.nodes, rid)
	for srcid := range esg.edges {
		delete(esg.edges[srcid], rid)
	}
	delete(esg.edges, rid) // delete out edges from rid
}

/*
why of the first note of hole must be FPDstEscAddrSrc/FPRootOutliveAddrSrc/FPParamLeak
Notice that fastpath only hashes through src and dst node.
If src and dst location are the same, then the hole will be overwritten.
*/
func (esg *escapeSubGraph) fastpath(src *location, newh hole) {
	dst := newh.dst
	srcN, dstN := esg.createOrGetNode(src), esg.createOrGetNode(dst)
	if flow, ok := esg.edges[srcN.id][dstN.id]; ok {
		if isEAReasonDebug() {
			_, file, line, _ := runtime.Caller(1)
			getLogger().Printf("fastpath: caller1: %s:%d, exist flow %v -> %v, add hole %s\n", path.Base(file), line, esg.b.explainLocPos(src), esg.b.explainLocPos(dst), strings.Join(newh.notes.whysString(), ", "))
		}
		flow.holes = append(flow.holes, newh)
	} else {
		if isEAReasonDebug() {
			_, file, line, _ := runtime.Caller(1)
			getLogger().Printf("fastpath: caller1: %s:%d, new flow %v -> %v, hole %s\n", path.Base(file), line, esg.b.explainLocPos(src), esg.b.explainLocPos(dst), strings.Join(newh.notes.whysString(), ", "))
		}
		esg.edges[srcN.id][dstN.id] = &escFlow{
			src:   srcN,
			dst:   dstN,
			holes: []hole{newh},
		}
	}
}

func (esg *escapeSubGraph) markLocReason(loc *location, reason escapeReason) {
	node := esg.createOrGetNode(loc)
	node.addReason(reason)
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
	esg.initSetUp()
	esg.dump(graphBatchRoot, phase2name[pInit])
	/*
		1. x -(d, y)-> heap
			remove heap(can be counted)
		2. back propogate from root to all source
			add node attribute(can be counted)
		3. for each source node, summarize all attributes(define priority)
	*/
	esg.removeHeap()
	esg.dump(graphBatchRoot, phase2name[pRemoveHeap])
	esg.backPropogate()
	esg.dump(graphBatchRoot, phase2name[pBackPropogate])
	esg.summarize(csvBatchRoot)
	esg.dump(graphBatchRoot, phase2name[pSummarize])
}

var graphRoot, csvRoot, esgLogPath string

var logger *log.Logger

func getLogger() *log.Logger {
	return logger
}

func initEAReason() {
	logger = log.New(os.Stdout, "ER: ", log.Lshortfile)
	graphRoot = path.Join(eaResultRoot(), "graphs", base.Ctxt.Pkgpath)
	csvRoot = path.Join(eaResultRoot(), "csv", base.Ctxt.Pkgpath)
	esgLogsRoot := path.Join(eaResultRoot(), "logs", base.Ctxt.Pkgpath)
	if base.Ctxt.Pkgpath == "main" {
		relativePath := path.Dir(flag.Arg(0))
		graphRoot = path.Join(graphRoot, relativePath)
		csvRoot = path.Join(csvRoot, relativePath)
		esgLogsRoot = path.Join(esgLogsRoot, relativePath)
	}
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

type phaseTy int

const (
	pInit phaseTy = iota
	pRemoveHeap
	pBackPropogate
	pSummarize
)

var phase2name [4]string = [4]string{
	"0init",
	"1removeheap",
	"2backPropogate",
	"summarize",
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

	// create svg asynchronously
	_esgDumpWaitGrp.Add(1)
	go func() {
		defer _esgDumpWaitGrp.Done()

		// write dot file
		n, err := writeToFile(dotfile, string(bs))
		if err != nil {
			fmt.Fprintf(os.Stderr, "write error %v\n", err)
		}
		if statDebug {
			fmt.Fprintln(os.Stderr, dotfile, "write bytes:", n)
		}

		if isESGDumpEnabled() {
			// set 30s context
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// dot with 30s context
			cmd := exec.CommandContext(ctx, "dot", "-Tsvg", dotfile, fmt.Sprintf("-o%s", svg))

			// run command
			if err := cmd.Run(); err != nil {
				// whether the command exceeds its ddl
				if ctx.Err() == context.DeadlineExceeded {
					fmt.Fprintf(os.Stderr, "Command timed out after 30 seconds: %s\n", cmd.String())
				} else {
					fmt.Fprintf(os.Stderr, "Command execution failed: %v\n", err)
				}
			}
		}
	}()

	if statDebug {
		// print node info
		for id, node := range esg.nodes {
			fmt.Fprintf(os.Stderr, "id %d: %s\n", id, node.Label())
			fmt.Fprintf(os.Stderr, "\treason %#b\n", node.reason)
		}
	}
}

/*
initSetUp prepares some fields for nodes and edges for faster computation
*/
func (esg *escapeSubGraph) initSetUp() {
	for _, toedges := range esg.edges {
		for _, flow := range toedges {
			for _, hole := range flow.holes {
				for note := hole.notes; note != nil; note = note.next {
					flow.whys = append(flow.whys, note.why)
				}
			}
		}
	}
}

func (esg *escapeSubGraph) mustNotInStack(node *escLocNode) bool {
	return node.loc == &esg.b.heapLoc ||
		node.reason.containAll(ERHeap) ||
		node.loc.isName(ir.PPARAMOUT) ||
		node.loc.isName(ir.PEXTERN) ||
		node.loc.isName(ir.PAUTOHEAP)
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
		if len(esg.edges[id]) == 0 && !esg.mustNotInStack(node) {
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
		if node.reason.containAll(ERRootStack | EROutliveRootStack) {
			node.reason = node.reason.del(EROutliveRootStack)
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

func (esg *escapeSubGraph) encodeLocLeakReason(loc *location, esc *leaks) (reason escapeReason, found bool) {
	id, ok := esg.loc2id[loc]
	if !ok {
		return 0, false
	}
	node := esg.nodes[id]
	esc.encodeReason(node.reason)
	return node.reason, true
}

func (esg *escapeSubGraph) findNode(label string) *escLocNode {
	for _, node := range esg.nodes {
		if strings.Contains(node.label, label) {
			return node
		}
	}
	return nil
}
