package main

import (
	"demo/er"
	"fmt"
	"strings"
)

// NodeInfo 表示结点信息
type NodeInfo struct {
	PackagePath    string `json:"package_path"`
	BatchNumber    uint64 `json:"batch_number"`
	NodeID         uint64 `json:"node_id"`
	NodeLocation   string `json:"node_location"`
	RootCauseCount uint16 `json:"root_cause_count"`
	Size           uint64 `json:"size"`
	Uid            uint64 `json:"uid"` // 唯一标识符，用于优化连接性能
}

// NodeRootCause 表示结点的逃逸根因
type NodeRootCause struct {
	Uid         uint64 `ch:"uid"`
	RootCauseID string `ch:"root_cause_id"`
}

func RootCauseEnumTypeString() string {
	strs := make([]string, 0, er.ERNumber)
	for i := range er.ERNumber {
		reason := er.EscapeReason(1 << i)
		strs = append(strs, fmt.Sprintf("'%s'", reason.ToEnumString()))
	}
	return fmt.Sprintf("Enum(%s = 0, %s)", strs[0], strings.Join(strs[1:], ", "))
}
