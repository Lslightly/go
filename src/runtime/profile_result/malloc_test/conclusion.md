以下结果为使用当前commit或者更早的commit(最早的没有breaking change的变更)获得的结果。

> 在循环前后添加 `b.ResetTimer()` `b.StopTimer()`

## 当前commit 结果

```
goos: linux
goarch: arm64
pkg: runtime
                     │ default.txt │              batch.txt              │
                     │   sec/op    │   sec/op     vs base                │
Malloc8-64             22.11n ± 0%   21.82n ± 0%   -1.31% (p=0.000 n=10)
Malloc16-64            38.79n ± 0%   33.76n ± 0%  -12.98% (p=0.000 n=10)
MallocTypeInfo8-64     28.49n ± 0%   31.37n ± 0%  +10.11% (p=0.000 n=10)
MallocTypeInfo16-64    38.19n ± 0%   39.57n ± 0%   +3.61% (p=0.000 n=10)
MallocLargeStruct-64   417.9n ± 1%   400.8n ± 1%   -4.10% (p=0.000 n=10)
geomean                52.27n        51.62n        -1.24%
```

添加了 `deductAssistCredit` 之后，对性能也基本没有影响。

## commit df0fac 对比结果

```
goos: linux
goarch: arm64
pkg: runtime
                    │ default.txt  │              batch.txt              │
                    │    sec/op    │   sec/op     vs base                │
MallocTypeInfo8-64    31.75n ± 10%   31.48n ± 0%        ~ (p=0.468 n=10)
MallocTypeInfo16-64   46.08n ±  2%   39.55n ± 0%  -14.17% (p=0.000 n=10)
geomean               38.25n         35.29n        -7.75%
```

这次`MallocTypeInfo16`又有下降。所以不是很稳定。可以尝试1)复测 2)添加assistGC


在MallocTypeInfo8-64, MallocTypeInfo16-64上还是有劣化，原因尚不清楚。

/cpu/classes/gc/mark/assist:cpu-seconds 指标下降明显1337.500m -> 2.760m，可能需要在batch关闭期间尝试assist。由于这些benchmark一起执行，处于同一个runtime，因此可能在执行到这两个函数的时候触发GC，发生了比较多的时间开销。

## trace

根据 [default.trace.out](./default.trace.out) 和 [batch.trace.out](./batch.trace.out)，基本上GC的并发标记和清扫步调是一致的。
