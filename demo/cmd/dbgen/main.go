package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math/bits"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"

	"demo/er"

	"github.com/goccy/go-yaml"
	"github.com/schollz/progressbar/v3"
)

var (
	enableProfile *bool = flag.Bool("profile", false, "enable profiling")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: dbgen <yaml config file, example.yaml for example>")
		flag.PrintDefaults()
	}
}

// 预编译一个正则表达式，匹配所有【非字母、非数字、非下划线】的字符
var illegalChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// CleanClickHouseIdentifier 将任意字符串转换为 ClickHouse 的合法非引用标识符
func CleanClickHouseIdentifier(input string) string {
	if input == "" {
		return "" // 处理空字符串情况
	}

	// 1. 将所有非法字符替换为下划线
	cleaned := illegalChars.ReplaceAllString(input, "_")

	// 2. 移除连续的下划线（可选：让名称更美观）
	// 匹配一个或多个下划线，替换为单个下划线
	cleaned = regexp.MustCompile(`_{2,}`).ReplaceAllString(cleaned, "_")

	// 3. 确保首字符合法（不以数字开头，或不以下划线开头，或两者）
	// ClickHouse允许下划线开头，但禁止数字开头

	// 检查首字符是否为数字
	firstChar := cleaned[0]
	if firstChar >= '0' && firstChar <= '9' {
		// 如果以数字开头，则在前面加上一个下划线作为前缀
		cleaned = "_" + cleaned
	}

	// 4. 将字符串转换为小写（可选：ClickHouse区分大小写，但推荐使用小写）
	// 统一为小写可以避免大小写混用的问题
	return strings.ToLower(cleaned)
}

func importStatRootData(statRoot string, fixname string, dbPrefix string) {
	fixcsvdir := filepath.Join(statRoot, fixname, "csv")
	dbName := CleanClickHouseIdentifier(dbPrefix + fixname)

	// 创建数据库连接
	ctx := context.Background()
	dbClient, err := OverwriteDB(ctx, "localhost:9000", dbName)
	if err != nil {
		fmt.Printf("创建数据库失败: %v\n", err)
		return
	}
	defer dbClient.Close()

	uidCounter := uint64(0)

	// 批量处理所有记录
	var nodeInfos []NodeInfo
	var allRootCauses []NodeRootCause

	var csvProgressCnt int
	filepath.WalkDir(fixcsvdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && d.Name() == "all.csv" {
			csvProgressCnt += 1
		}
		return nil
	})

	bar := progressbar.Default(int64(csvProgressCnt), fmt.Sprintf("collect csv for %s", fixname))

	// 使用 WalkDir 遍历目录
	err = filepath.WalkDir(fixcsvdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理 all.csv 文件
		if !d.IsDir() && d.Name() == "all.csv" {
			defer bar.Add(1)
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
				nodeInfos = append(nodeInfos, nodeInfo)

				// 构建 NodeRootCause 列表并添加到总列表中
				for i := range er.ERNumber {
					r := er.EscapeReason(1 << i)
					if er.EscapeReason(reason)&r != 0 {
						allRootCauses = append(allRootCauses, NodeRootCause{
							Uid:         uidCounter,
							RootCauseID: r.ToEnumString(),
						})
					}
				}

				uidCounter++
			}

		}
		return nil
	})
	if err != nil {
		log.Fatalf("遍历目录时出错: %v\n", err)
	}
	// 批量插入数据
	if len(nodeInfos) > 0 {
		if err := dbClient.InsertNodeInfoAndRootCausesBatch(ctx, nodeInfos, allRootCauses, dbName); err != nil {
			log.Fatal(err)
		}
	}
}

type config struct {
	StatRoot string   `yaml:"statRoot"`
	DBPrefix string   `yaml:"dbPrefix"`
	Cases    []string `yaml:"cases"`
}

func paritialProfile() *os.File {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	cpuprofile, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(cpuprofile)
	go func() {
		defer cpuprofile.Close()
		sig := <-sigChan
		fmt.Printf("Received Signal %v. Stopping profiling and exiting.\n", sig)
		pprof.StopCPUProfile()
		os.Exit(1)
	}()
	return cpuprofile
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
	if *enableProfile {
		cpuprof := paritialProfile()
		defer func() {
			pprof.StopCPUProfile()
			cpuprof.Close()
		}()
	}
	for _, fixname := range cfg.Cases {
		importStatRootData(cfg.StatRoot, fixname, cfg.DBPrefix)
		log.Println(fixname, "imported")
	}
}
