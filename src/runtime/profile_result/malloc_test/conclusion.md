用benchstat对比效果如下，其中old为default配置，new为添加了批次分配优化的配置。

```
goos: linux
goarch: arm64
pkg: runtime
                     │   old.txt   │               new.txt                │
                     │   sec/op    │    sec/op     vs base                │
Malloc8-64             22.02n ± 0%   19.07n ±  1%  -13.37% (p=0.000 n=10)
Malloc16-64            38.49n ± 0%   29.90n ±  2%  -22.33% (p=0.000 n=10)
MallocTypeInfo8-64     28.50n ± 0%   28.05n ±  1%   -1.58% (p=0.000 n=10)
MallocTypeInfo16-64    38.12n ± 0%   35.62n ±  1%   -6.55% (p=0.000 n=10)
MallocLargeStruct-64   402.5n ± 0%   366.1n ± 17%   -9.03% (p=0.021 n=10)
geomean                51.73n        46.12n        -10.85%
```

虽然从循环malloc的时间来看降低了，但是测试的总体时间上升了。

default配置的测试执行时间是433.031s，批次分配优化配置的时间是538.590s

