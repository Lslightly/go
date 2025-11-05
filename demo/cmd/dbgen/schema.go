package main

import (
	"context"
	"fmt"
)

// OverwriteDB 删除现有数据库并创建新的演示数据库和表结构
func OverwriteDB(ctx context.Context, hostPort, dbName string) (*DBClient, error) {
	err := func() error {
		admin, _ := NewAdminClient(ctx, hostPort)
		defer admin.Close()
		// 1. 删除现有的数据库（如果存在）
		if err := admin.conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)); err != nil {
			return fmt.Errorf("删除数据库失败: %w", err)
		}

		// 2. 创建新的数据库
		if err := admin.conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
			return fmt.Errorf("创建数据库失败: %w", err)
		}

		// 3. 在新数据库中创建 node_info 表
		nodeInfoDDL := fmt.Sprintf(`
    CREATE TABLE %s.node_info (
        package_path String,
        batch_number UInt64,
        node_id UInt64,
        node_location String,
        root_cause_count UInt16,
        uid UInt64
    ) ENGINE = MergeTree()
    ORDER BY (uid)
    SETTINGS index_granularity = 8192;
    `, dbName)
		if err := admin.conn.Exec(ctx, nodeInfoDDL); err != nil {
			return fmt.Errorf("创建 node_info 表失败: %w", err)
		}

		// 5. 在新数据库中创建 node_root_causes 表
		nodeRootCausesDDL := fmt.Sprintf(`
    CREATE TABLE %s.node_root_causes (
        uid UInt64,
        root_cause_id %s,
    ) ENGINE = MergeTree()
    ORDER BY (uid, root_cause_id)
    SETTINGS index_granularity = 8192;
    `, dbName, RootCauseEnumTypeString())
		if err := admin.conn.Exec(ctx, nodeRootCausesDDL); err != nil {
			return fmt.Errorf("创建 node_root_causes 表失败: %w", err)
		}

		return nil
	}()
	if err != nil {
		return nil, err
	}
	client, err := NewAppClient(ctx, hostPort, dbName)
	if err != nil {
		return nil, err
	}
	return client, nil
}
