# 批次分配

## 问题

为了保证对象在分配之后的零初始化让GC感知，需要添加屏障操作。但是屏障操作会阻塞后续的store指令，影响mallocgc热点路径的执行效率，降低程序性能。例如：

因此，希望通过识别批次分配的模式，通过共享屏障的方式，减少屏障的执行时间，并复用其他相似逻辑，加快分配速度，提升程序性能。

## 动机案例

```go
s := []*ObjT{}
for {
    p := new(ObjT)
    s = append(s, p)
}
```

在上述样例中，`new`操作会被Go编译器重写为`mallocgc`函数，每次 `mallocgc` 函数在分配完对象以及零初始化之后，都会通过 `publicationBarrier()` 函数调用执行 `DMB ST` 全屏障指令，其代码模式如下所示。该指令会使得屏障前的store指令一定比屏障后的store指令先提交，从而保证新分配的对象在GC可见前一定已经完全零初始化，不会让GC在标记阶段读到指针类型的旧值，GC也不会根据错误的旧值继续标记其他活跃对象。

```go
s := []*ObjT{}
for {
    p := domalloc(ObjT)
    publicationBarrier()
    s = append(s, p)
}
```

但是在以上 `for` 循环中，屏障指令在每次 `mallocgc` 之后都会调用。这导致了性能问题，如：

## 预期优化

```go
s := []*ObjT{}
mp.mallocing = 1
for {
    p := domalloc(ObjT)
    s = append(s, p)
}
publicationBarrier()
mp.mallocing = 0
```

通过将 `publicationBarrier()` 提到循环之外，让多个分配共享屏障的方式，实现零初始化对GC的批量通知。

### 优化风险

由于目前对以下几方面的认知不足，以上优化是否有收益仍待商榷：
1. 如何保证延迟屏障对Go GC标记的正确性不会有影响
   1. Go GC标记是否有safe point，如果不需要safe point，如何保证正确性？
   2. `mp.mallocing`和Go GC标记是否有关？
2. `mallocgc`对Go调度器的性能影响
3. `mallocgc`对Go GC标记的性能影响

## 模式识别

动机案例中提到的模式在Go语言项目中存在，例如：

### 优化模式约束

由于目前采用 `runtime.BatchEnter` 和 `runtime.BatchExit` 两个函数，因此在这个上下文之间，不能有return，退出该上下文的情况，或者至少应该通过 `defer runtime.BatchExit()`的形式保证可以退出batch模式。如果是处在某个for循环内，由于控制流退出这个for循环，即退出上下文，那么从代码变换的角度可能就不好做。

比如

```
for {
    BatchEnter()
    for {
        break
        continue
        return
    }
    BatchExit()
}
```

如果采用函数调用嵌套defer的方式，就会修改为

```go
for {
    c := runtime.Batch(func() {
        for {
            return BREAK // break
            return CONTINUE // continue
            return RETURN // return
        }
    })
    if c == BREAK {
        break
    }
    if c == CONTINUE {
        continue
    }
    if c == RETURN {
        return
    }
}
```

会比较复杂。所以需要对for循环做限制
