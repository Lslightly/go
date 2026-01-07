# rclone benchmark for malloc

以下均为x64上的测试的结果

| name                        | mallocgc(%)(GOGC=on) | significant                                          | mallocgc(%)(GOGC=off) | significant                                                                                |
| --------------------------- | -------------------- | ---------------------------------------------------- | --------------------- | ------------------------------------------------------------------------------------------ |
| BenchmarkQuickXorHash       | 1.65%                | xorBytes(71.07%)                                     | 0                     | xorBytes(72.95%)                                                                           |
| BenchmarkCheckParents       | 1.30%                | gcDrain(82.63%)                                      | 13.85%                | memclrNoHeapPointers(8.82%)                                                                |
| BenchmarkSorterExt          | 8.35%                | gcDrain(27.16%)                                      | 13.16%                | mallocgcSmallScanNoHeader(6.56%), mallocgcSmallNoscan(5.37%),  memclrNoHeapPointers(5.38%) |
| BenchmarkPresent            | 5.87%                | gcDrain(23.46%)                                      | 8.95%                 | runtime.tracebackPCs(39.30%)                                                               |
| BenchmarkPresent100         | 2.42%                | runtime.tracebackPCs(26.61%)                         | 2.56%                 | runtime.tracebackPCs(29.91%)                                                               |
| BenchmarkOneDriveReplaceNew | 10.67%               | gcDrain(25.28%)                                      | 13.08%                | memclrNoHeapPointers(5.38%)                                                                |
| BenchmarkOneDriveReplaceOld | 8.24%                | gcDrain(10.59%)                                      | 11.72%                | regexp.(*Regexp).tryBacktrace(41.28%)                                                      |
| BenchmarkOneDriveRestoreNew | 9.82%                | gcDrain(15.34%), encoder.MultiEncoder.Decode(75.46%) | 12.88%                | encoder.MultiEncoder.Decode(93.18%)                                                        |
| BenchmarkOneDriveRestoreOld | 2.65%                | mapaccess2_fast32(56.29%), gcDrain(14.57%)           | 3.17%                 | mapaccess2_fast32(61.11%)                                                                  |

> significant 一般是gcDrain或者flat值(单看某个函数内执行时间，不将这个函数调用的函数算在其中的时间)较大的。

在 `GOGC=off` 的情况下，`BenchmarkCheckParents` 的mallocgc时间开销就显现出来了。

`BenchmarkAllocFree`, `BenchmarkAllocFreeWithLotsOfAllocations`, `BenchmarkAllocFreeForLotsOfAllocations` 基本上都在做mmap, munmap系统调用。

从占比来看，比较适合的benchmark如下：

```csv
BenchmarkCheckParents,GOGCoff
BenchmarkSorterExt,GOGCoff
BenchmarkOneDriveReplaceNew,GOGCoff
BenchmarkOneDriveReplaceOld,GOGCoff
BenchmarkOneDriveRestoreNew,GOGCoff
```

