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

EAResultRoot / "escape_reason_stat.txt"
EAResultRoot / "all.csv"
EAResultRoot / "escape_reason_dag.png"
三个文件是总计结果