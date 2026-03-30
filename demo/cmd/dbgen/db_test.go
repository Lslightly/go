package main

import (
	"context"
	"fmt"
	"log"

	"demo/er"
)

// ExampleUsage 展示如何使用该客户端
func ExampleDBClient_InsertNodeInfoAndRootCauses() {
	dbName := "demo_example"
	ctx := context.Background()
	client, err := OverwriteDB(ctx, "127.0.0.1:9000", dbName)
	if err != nil {
		log.Fatalf("Failed to overwrite DB %s: %v", dbName, err)
	}
	defer client.Close()

	// 示例数据
	nodeInfo := NodeInfo{
		PackagePath:    "example.com/mypackage",
		BatchNumber:    1,
		NodeID:         1001,
		NodeLocation:   "main.go:42",
		RootCauseCount: 2,
		Size:           1024,
		Uid:            0,
	}

	rootCauses := []NodeRootCause{
		{
			Uid:         0,
			RootCauseID: er.ERHeap.ToEnumString(),
		},
		{
			Uid:         0,
			RootCauseID: er.ERGo.ToEnumString(),
		},
	}

	if err := client.InsertNodeInfoAndRootCauses(ctx, nodeInfo, rootCauses); err != nil {
		log.Fatalf("Failed to insert data: %v", err)
	}
	fmt.Println("Data inserted successfully!")
	// Output: Data inserted successfully!
}
