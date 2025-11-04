package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/bits"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"demo/er"

	"github.com/goccy/go-yaml"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: dbgen <yaml config file, example.yaml for example>")
		flag.PrintDefaults()
	}
}

func importStatRootData(statRoot string, fixname string, dbPrefix string) {
	fixcsvdir := filepath.Join(statRoot, fixname, "csv")

	// 创建数据库连接
	ctx := context.Background()
	dbClient, err := OverwriteDB(ctx, "localhost:9000", dbPrefix+fixname)
	if err != nil {
		fmt.Printf("创建数据库失败: %v\n", err)
		return
	}
	defer dbClient.Close()

	uidCounter := uint64(0)

	// 使用 WalkDir 遍历目录
	err = filepath.WalkDir(fixcsvdir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理 all.csv 文件
		if !d.IsDir() && d.Name() == "all.csv" {
			// 解析路径获取 pkgpath 和 batch number
			relPath, err := filepath.Rel(fixcsvdir, path)
			if err != nil {
				return err
			}

			parts := strings.Split(relPath, string(os.PathSeparator))

			// pkgpath 是除了最后两个部分之外的所有部分，batch number 是倒数第二个部分，因为最后一部分是 "all.csv"
			if len(parts) < 3 {
				return fmt.Errorf("invalid path format: %s", relPath)
			}
			batchStr := parts[len(parts)-2] // 倒数第二个部分是 batch number
			pkgpath := strings.Join(parts[:len(parts)-2], string(os.PathSeparator))

			batchNumber, err := strconv.ParseUint(batchStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid batch number: %s", batchStr)
			}

			// 读取 CSV 文件
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			reader := csv.NewReader(file)
			records, err := reader.ReadAll()
			if err != nil {
				return err
			}

			// 跳过表头
			for i, record := range records {
				if i == 0 {
					continue // 跳过表头
				}

				if len(record) < 3 {
					log.Fatalf("line %d is incomplete. expected 3, actual %d\n", i+1, len(records))
				}

				nodeID, err := strconv.ParseUint(record[0], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid nodeid: %s", record[0])
				}

				reason, err := strconv.ParseInt(record[1], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid reason: %s", record[1])
				}

				loc := record[2]

				// 计算 RootCauseCount (reason 中 1 的位数)
				rootCauseCount := uint16(bits.OnesCount64(uint64(reason)))

				// 构建 NodeInfo
				nodeInfo := NodeInfo{
					PackagePath:    pkgpath,
					BatchNumber:    batchNumber,
					NodeID:         nodeID,
					NodeLocation:   loc,
					RootCauseCount: rootCauseCount,
					Uid:            uidCounter,
				}

				// 构建 NodeRootCause 列表
				var rootCauses []NodeRootCause
				for i := range er.ERNumber {
					r := er.EscapeReason(1 << i)
					if er.EscapeReason(reason)&r != 0 {
						rootCauses = append(rootCauses, NodeRootCause{
							Uid:                  uidCounter,
							RootCauseID:          r,
							RootCauseDescription: r.String(),
						})
					}
				}

				// 插入数据
				if err := dbClient.InsertNodeInfoAndRootCauses(ctx, nodeInfo, rootCauses); err != nil {
					return fmt.Errorf("failed to insert data: %w", err)
				}

				uidCounter++
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("遍历目录时出错: %v\n", err)
	}
}

type config struct {
	StatRoot string   `yaml:"statRoot"`
	DBPrefix string   `yaml:"dbPrefix"`
	Cases    []string `yaml:"cases"`
}

func main() {
	flag.Parse()
	cfgFile := flag.Arg(0)
	buf, err := os.ReadFile(cfgFile)
	if err != nil {
		log.Fatalf("Read file %s: %v", cfgFile, err)
	}
	var cfg config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	for _, fixname := range cfg.Cases {
		importStatRootData(cfg.StatRoot, fixname, cfg.DBPrefix)
		log.Println(fixname, "imported")
	}
}
