# profilealloc

在 `malloc.go` 中，`profilealloc` 函数调用的条件如下：

```go
c.nextSample -= int64(size)
if c.nextSample < 0 || MemProfileRate != c.memProfRate {
	profilealloc(mp, x, size)
}
```

`nextSample  int64` trigger heap sample after allocating this many bytes
`memProfRate int` cached mem profile rate, used to detect changes
`scanAlloc   uintptr` bytes of scannable heap allocated

`MemProfileRate` 每分配 `MemProfileRate` bytes 采样一个分配。处理 memory profile 的工具假设 `MemProfileRate` 是一个常数。如果程序要改这个数，最好只改一次，而且越早越好。当用户程序修改了 `MemProfileRate` 之后，会强制进行一次 profiling。

`c.nextSample` 根据已经分配的 size 判断是否要执行一次 profiling。

`c.nextSample = nextSample()` 会按照 `MemProfileRate` 为期望的指数分布生成随机数，以保证期望按照每 `MemProfileRate` bytes 分配，就实现一次采样。本质上来说， `c.nextSample < 0` 的事件就是带参数 `MemProfileRate` 的泊松过程。平均分配 `MemProfileRate` 字节一次 profiling。

`c.nextSample` 和 `MemProfileRate != c.memProfile` 会对全局变量以及 `c.nextSample` 做 load 操作，可以放在屏障前。
- 即使对 `MemProfileRate` 有并发的修改，也不影响。
- `c.nextSample` 不会有并发的写入

# mProf_Malloc



