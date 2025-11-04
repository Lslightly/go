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
	ERNumber    = iota
)

var escapeReasonToString = map[EscapeReason]string{
	ERHeap:             "heap",
	ERStar:             "*x",
	ERRootStack:        "stack",
	EROutliveRootStack: "outlive_stack",
	ERClosure:          "closure",
	ERGo:               "go",
	ERDefer:            "defer",
	ERReturn:           "return",
	ERParamLeak:        "param_leak",
	ERMapIdx:           "map_index",
	ERSliceIdx:         "slice_index",
	ERComm:             "communication",
	ERAppend:           "append",
	ERCopy:             "copy",
	ERMust:             "must_heap",
}

// String 方法用于将枚举值转换为字符串描述
func (e EscapeReason) String() string {
	if str, exists := escapeReasonToString[e]; exists {
		return str
	}
	return fmt.Sprintf("unknown_%d", e)
}
