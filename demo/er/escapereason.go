// Must sync with src/cmd/compile/internal/escape/stat_subgraph.go
package er

import (
	"fmt"
)

// bitstring for escape reason
type EscapeReason int64

const (
	ERHeap             EscapeReason = 1 << iota // root is heap
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

var escapeReasonToEnumString = map[EscapeReason]string{
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

func (e EscapeReason) ToEnumString() string {
	if str, exists := escapeReasonToEnumString[e]; exists {
		return str
	}
	return fmt.Sprintf("unknown_%d", e)
}
