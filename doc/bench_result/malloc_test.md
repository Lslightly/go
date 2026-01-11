# malloc_test.go Benchmark

publicationBarrier的开销分析见 [issue comment](https://github.com/golang/go/issues/63640#issuecomment-3661284210)

- [nops_mask.out](../../src/runtime/profile_result/malloc_test/nops_mask.out)
- [nodmb.out](../../src/runtime/profile_result/malloc_test/nodmb.out)
- [cpu.out](../../src/runtime/profile_result/malloc_test/cpu.out)

nops_mask的代码如下:

```go
noploop(120)
publicationBarrier()
// Account for barrier latency at
// this nop, not the next one.
nop()
span.freeIndexForScan = span.freeindex
if writeBarrier.enabled {
    gcmarknewobject(span, uintptr(x))
}
c.nextSample -= int64(size)
if c.nextSample < 0 || MemProfileRate != c.memProfRate {
    profilealloc(mp, x, size)
}
```

default

```go
publicationBarrier()
span.freeIndexForScan = span.freeindex
```

| abbreviation | config                                                                              | expectation for the config                                                                                                                         | statements following `publicationBarrier`           | profile file            | 
| ------------ | ----------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------- | ----------------------- |
| nops_mask    | `noploop` + `publicationBarrier` + `nop` + `span.freeIndexForScan = span.freeindex` | The latency for `publicationBarrier` should be masked by `noploop` and the time for `nop`+`span.freeIndexForScan = span.freeindex` should be less. | `nop`+`span.freeIndexForScan = span.freeindex`      | nops_mask.out | 
| nodmb        | `span.freeIndexForScan = span.freeindex`                                            | The time for `span.freeIndexForScan = span.freeindex` should be less.                                                                              | `span.freeIndexForScan = span.freeindex`(no DMB ST) | nodmb.out               |
| default      | `publicationBarrier` + `span.freeIndexForScan = span.freeindex`                     | The time of `span.freeIndexForScan = span.freeindex` is large and can't be ignored in the view of `mallogcSmallNoscan`                             | `span.freeIndexForScan = span.freeindex`            | cpu.out                 |

time for each config

mallocgcSmallNoscan

| cum                                         | nops_mask | nodmb | default |
| ------------------------------------------- | --------- | ----- | ------- |
| mallocgcSmallNoscan                         | 1.18s     | 990ms | 1.22s   |
| noploop(120)                                | 690ms     | -     | -       |
| publicationBarrier                          | 30ms      | -     | 20ms    |
| nop                                         | 190ms     | -     | -       |
| span.freeIndexForScan = span.freeindex      | 20ms      | 50ms  | 500ms   |
| writeBarrier.enabled                        | 0         | 0     | 0       |
| gcmarknewobject                             | 0         | 10ms  | 10ms    |
| c.nextSample -= int64(size) if c.nextSample < 0 ... | 10ms      | 20ms  | 10ms    |
| profilealloc                                | 0         | 0     | 20ms    |

mallocgcSmallScanNoHeader

| cum                                         | nops_mask | nodmb | default |
| ------------------------------------------- | --------- | ----- | ------- |
| mallocgcSmallScanNoHeader                   | 2.47s     | 1.99s | 2.29s   |
| noploop(140)                                | 1.62s     | -     | -       |
| publicationBarrier                          | 0         | -     | 30ms    |
| nop                                         | 200ms     | -     | -       |
| span.freeIndexForScan = span.freeindex      | 20ms      | 10ms  | 820ms   |
| writeBarrier.enabled                        | 0         | 0     | 0       |
| gcmarknewobject                             | 10ms      | 50ms  | 30ms    |
| c.nextSample -= int64(size) if c.nextSample < 0 ... | 70ms      | 50ms  | 0       |
| profilealloc                                | 0         | 0     | 20ms    |


问题：

noploop时间占到1.62s，可能会影响其他代码的统计？不知道代码行做时间统计的原理是什么。

## profilealloc判断条件前提以及writeBarrier.enabled load指令前提对性能产生的影响

profilealloc判断条件和writeBarrier.enabled前提之后的代码片段为：

```go
wbEnabled := writeBarrier.enabled // eager calc
c.nextSample -= int64(size)       // eager calc
doSample := c.nextSample < 0 || MemProfileRate != c.memProfRate // eager calc
publicationBarrier()
span.freeIndexForScan = span.freeindex
if wbEnabled {
    gcmarknewobject(span, uintptr(x))
}
if doSample {
    profilealloc(mp, x, size)
}
mp.mallocing = 0
```

变更前代码片段为：

```go
publicationBarrier()
span.freeIndexForScan = span.freeindex
if writeBarrier.enabled {
    gcmarknewobject(span, uintptr(x))
}
c.nextSample -= int64(span.elemsize)
if c.nextSample < 0 || MemProfileRate != c.memProfRate {
    profilealloc(mp, x, span.elemsize)
}
mp.mallocing = 0
```

变更后的profiling结果见 [eager_calc.out](../../src/runtime/profile_result/malloc_test/eager_calc.out)
变更前的profiling结果见 [cpu.out](../../src/runtime/profile_result/malloc_test/cpu.out)

开销对比

mallocgcSmallScanNoHeader

| cum                                       | eager_calc | default |
| ----------------------------------------- | ---------- | ------- |
| mallocgcSmallScanNoHeader                 | 2.36s      | 2.21s   |
| writeBarrier.enabled                      | 0          | 0       |
| c.nextSample -= ..., c.nextSample < 0 ... | 10ms       | 40ms    |
| publicationBarrier                        | 20ms       | 30ms    |
| span.freeIndexForScan = span.freeindex    | 890ms      | 850ms   |
| gcmarknewobject                           | 0        | 40ms    |
| profilealloc                              | 0        | 10ms    |

提前后基本没有提升，甚至 `span.freeIndexForScan = span.freeindex`还需要更多时间了

mallocgcSmallNoscan

| cum                                       | eager_calc | default |
| ----------------------------------------- | ---------- | ------- |
| mallocgcSmallNoscan                       | 1.09s      | 1.23s   |
| writeBarrier.enabled                      | 0          | 0       |
| c.nextSample -= ..., c.nextSample < 0 ... | 10ms       | 40ms    |
| publicationBarrier                        | 0          | 30ms    |
| span.freeIndexForScan = span.freeindex    | 450ms      | 570ms   |
| gcmarknewobject                           | 0          | 10ms    |
| profilealloc                              | 10ms       | 10ms    |

在 `mallocgcSmallNoscan` 中 `span.freeIndexForScan = span.freeindex`有降低

