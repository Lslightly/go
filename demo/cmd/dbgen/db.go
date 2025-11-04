package main

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// DBClient 封装了与 ClickHouse 的连接
type DBClient struct {
	conn driver.Conn
}

// 创建管理连接（不依赖具体数据库）
func NewAdminClient(ctx context.Context, hostPort string) (*DBClient, error) {
	// 连接到默认数据库
	return newDBClient(ctx, hostPort, "default")
}

// 创建业务连接（使用特定数据库）
func NewAppClient(ctx context.Context, hostPort, dbName string) (*DBClient, error) {
	return newDBClient(ctx, hostPort, dbName)
}

// newDBClient 创建一个连接到指定数据库的客户端
func newDBClient(ctx context.Context, hostPort, dbName string) (*DBClient, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{hostPort},
		Auth: clickhouse.Auth{
			Database: dbName,
			Username: "default",
			Password: "",
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	// 检查连接
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return &DBClient{conn: conn}, nil
}

// Close 关闭数据库连接
func (db *DBClient) Close() error {
	return db.conn.Close()
}

// InsertNodeInfoAndRootCauses 将结点信息和其所有根因插入到 ClickHouse
func (db *DBClient) InsertNodeInfoAndRootCauses(ctx context.Context, nodeInfo NodeInfo, rootCauses []NodeRootCause) error {
	// 在 ClickHouse 的 Go 驱动中，连接本身可以作为事务使用。
	tx := db.conn
	defer func() {
		// 连接不由此函数关闭，所以这里不执行操作
	}()

	// 插入 node_info 表
	batch, err := tx.PrepareBatch(ctx, "INSERT INTO node_info")
	if err != nil {
		return fmt.Errorf("failed to prepare batch for node_info: %w", err)
	}
	if err := batch.Append(
		nodeInfo.PackagePath,
		nodeInfo.BatchNumber,
		nodeInfo.NodeID,
		nodeInfo.NodeLocation,
		nodeInfo.RootCauseCount,
		nodeInfo.Uid,
	); err != nil {
		return fmt.Errorf("failed to append to node_info batch: %w", err)
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send node_info batch: %w", err)
	}

	// 批量插入 node_root_causes 表
	if len(rootCauses) > 0 {
		batch, err = tx.PrepareBatch(ctx, "INSERT INTO node_root_causes")
		if err != nil {
			return fmt.Errorf("failed to prepare batch for node_root_causes: %w", err)
		}
		for _, cause := range rootCauses {
			if err := batch.Append(
				nodeInfo.Uid,
				int64(cause.RootCauseID),
				cause.RootCauseDescription,
			); err != nil {
				return fmt.Errorf("failed to append to node_root_causes batch: %w", err)
			}
		}
		if err := batch.Send(); err != nil {
			return fmt.Errorf("failed to send node_root_causes batch: %w", err)
		}
	}

	// 在 ClickHouse 中，PrepareBatch.Send() 即是原子的写入操作。
	// 因此，我们不需要显式的 Commit。如果所有 Send() 都成功，则认为操作成功。
	return nil
}
