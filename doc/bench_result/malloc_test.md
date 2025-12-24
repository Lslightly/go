# malloc_test.go Benchmark

publicationBarrier的开销分析见 [issue comment](https://github.com/golang/go/issues/63640#issuecomment-3661284210)

- [batch_share_barrier.out](../../src/runtime/profile_result/batch_share_barrier.out)
- [nodmb.out](../../src/runtime/profile_result/nodmb.out)
- [cpu.out](../../src/runtime/profile_result/cpu.out)

| abbreviation | config                                                                              | expectation for the config                                                                                                                         | statements following `publicationBarrier`           | profile file            | 
| ------------ | ----------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------- | ----------------------- |
| nops_mask    | `noploop` + `publicationBarrier` + `nop` + `span.freeIndexForScan = span.freeindex` | The latency for `publicationBarrier` should be masked by `noploop` and the time for `nop`+`span.freeIndexForScan = span.freeindex` should be less. | `nop`+`span.freeIndexForScan = span.freeindex`      | batch_share_barrier.out | 
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

noploop时间占到2.31s，
