# 逃逸根因分析

## 配置

两大步配置：编译器配置和编译源文件配置

### 编译器配置

1. 先在master分支，使用 [src/make.bash](../../src/make.bash) 构建工具链
2. 使用 [toolstash](https://golang.org/x/tools/cmd/toolstash) 工具备份工具链

```bash
# 1.24.0版本go对应的toolstash最新版本是v0.42.0，如果不想自动下载更新版本go的话，可以将`latest`修改为`v0.42.0`
go install golang.org/x/tools/cmd/toolstash@latest
toolstash save # 保存在$GOROOT/pkg/toolstash下，通过toolstash restore恢复
```

3. `go install cmd/compile`编译包含逃逸根因分析的编译器

### 编译选项与逃逸根因分析开启

有两种方式，建议使用第1种。

1. 环境变量

`EAReasonStat`是否开启逃逸根因分析(开启方式为`EAReasonStat=1`, `=0`不开启)
`EAResultRoot`逃逸根因分析的结果存储位置

`EAReasonDebug`是否打印调试信息。(开启方式为`EAReasonDebug=1`, `=0`不开启)
`ESGDumpEnabled`是否通过`dot -Tsvg`生成svg文件。（如果图较大，会占用较多时间，默认关闭）

2. gcflags

三个flag

`-eareasonstat`指定是否开启逃逸根因分析，添加即开启
`-earesultroot`指定逃逸根因分析结果目录
`-eareasondebug`是否打印调试信息，添加即开启

## 产物位置

在指定`EAResultRoot`之后，在root下生成以下3个目录
- `graphs`
- `csv`
- `logs`

每个目录下按照package path组织

main package的package path会修改为main package下文件的绝对目录

