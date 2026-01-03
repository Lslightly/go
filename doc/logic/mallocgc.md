# mallocgc

`mallocgc`的逻辑，会针对不同的malloc size调用更细粒度的不同分配处理函数，其分支逻辑如下图所示：

![](../figures/mallocgc_branches.png)

分配函数总共分为5类：Tiny, SmallNoscan, SmallScanNoHeader, SmallScanHeader, Large

其中:
- Tiny: size < maxTinySize (16 bytes) 且对象不包含指针(typ == nil || !typ.Pointers())
- SmallNoscan: size <= maxSmallSize (32kB) 且对象不包含指针(typ == nil || !typ.Pointers()) 且 size >= maxTinySize
- SmallScanNoHeader: size <= maxSmallSize (32kB) 且对象包含指针(typ != nil && typ.Pointers()) 且 heapBitsInSpan(size) 为 true
- SmallScanHeader: size <= maxSmallSize (32kB) 且对象包含指针(typ != nil && typ.Pointers()) 且 heapBitsInSpan(size) 为 false
- Large: size > maxSmallSize (32kB)

每个分配函数之后都会有 `publicationBarrier` 屏障函数调用。屏障函数内部针对不同平台做了不同的实现。对于x86平台来说，由于x86平台采用了Total Store Order(TSO)内存模型，store指令之间相互有序，因此不需要在将对象零初始化以及通知GC当前空闲的内存分配slot之间添加同步指令，`publicationBarrier`函数实现为只有一条 return 指令。而对于 aarch64 这种弱内存序模型，store 指令之间可能会乱序执行，此时就通过 `DMB ST` 屏障指令保证不同 store 指令之间保证按照程序序提交。

不同分配函数在屏障之前的逻辑如下所示：

Tiny:

![](../figures/mallocgcTiny.png)

SmallNoscan

![](../figures/mallocSmallNoscan.png)

SmallScanHeader

![](../figures/mallocgcSmallScanHeader.png)

SmallScanNoHeader

![](../figures/mallocgcSmallScanNoHeader.png)

Large

![](../figures/mallocgcLarge.png)
