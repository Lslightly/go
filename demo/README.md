## 文件作用说明

run.py

```
usage: run [-h] [--build] [--demo]

options:
  -h, --help  show this help message and exit
  --build     Whether to build the project
  --demo      Whether to run demo only
```

stat_root: 存放其他项目的逃逸根因统计结果
fixtures: 存放测试项目

### 逃逸根因统计的一般流程

1. 使用`python run.py --build`编译项目，在`/data/github_go/escapeReasonStatRoot`目录下生成统计数据。
2. 使用`go run ./cmd/dbgen ./cmd/dbgen/bench.yaml` 将这些统计数据添加到数据库中
3. 使用`./cmd/dbgen/sql`下的SQL查询数据库。

