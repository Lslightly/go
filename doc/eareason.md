# 逃逸原因

逃逸原因主要分为以下16类：

```
	ERHeap             escapeReason = 1 << iota // root is heap
	ERStar                                      // root is *x
	ERRootStack                                 // root in stack
	EROutliveRootStack                          // outlived by root in stack
	ERClosure                                   // root is closure
	ERGo                                        // root is go
	ERDefer                                     // root is defer
	ERReturn                                    // l is returned
	ERParamLeak                                 // param leak
	ERCallArgument                              // root is call argument
	ERMapIdx                                    // m[index]
	ERSliceIdx                                  // s[index]
	ERComm                                      // communication by send
	ERAppend                                    // append
	ERCopy                                      // copied slice
	ERMust                                      // must be heap allocated
```

`ERRootStack`表示由于违反不变式2(指针栈对象的指针生命期不能超过该栈对象)导致的逃逸
`ERClosure`是由于闭包逃逸导致对象逃逸
`ERParamLeak`是由于参数在被调用者中逃逸导致逃逸
`ERMapIdx`是由于作为map的key导致逃逸
`ERMust`是必须在堆上分配的，比如makeslice大小超过了限制

## 统计结果

- EAResultRoot / "escape_reason_stat.txt"
- EAResultRoot / "all.csv"
- EAResultRoot / "escape_reason_dag.png"
三个文件是总计结果

## 根因产生位置

ERHeap:
- FPDstEscAddrSrc
- FPRootOutliveAddrSrc
- 流向heap结点的源节点都会标记ERHeap
- 如果outlive(root, edge.src)，且root有attrEscapes，则标记为ERHeap
- 

## 数据库schema

数据维度：

- package path
- batch number, 一个batch内可能分析多个函数
- 结点id
- 结点逃逸根因
  - 一个结点的逃逸根因可能有多个
- 结点逃逸根因数量
  - 根据结点逃逸根因统计的结果，方便加速查询
- 结点位置
  - 一个结点只有一个位置。

package path+batch number+结点id可以唯一确定一个结点

设计两张表 schema，一张表用于存储结点根因，一张表存储其他信息。

数据库可能采用列式数据库。用python还是用Go api好一点？逃逸根因常量我可以在生成这些根因的Go程序中拷贝常量。

