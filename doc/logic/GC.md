# GO GC

GC bitmap gcmask用于标记对象内每8字节内存是否包含指针

gcmarkBits用于确定当前对象是否已被扫描。0表示白色，1表示黑色。如果对象在工作队列中，则表示灰色，等待扫描子对象。

## goref

[goref design priciple](https://github.com/cloudwego/goref/blob/main/docs/principle.md) 设计原则，其中包含了对GC的部分描述。

## GC整体流程与调度

5个阶段：
1. 清扫终止阶段。
   1. Stop the world。让所有P到达GC安全点。
   2. 清扫未被清扫的span。
2. 标记阶段
   1. 将 `gcphase` 从 `_GCoff` 切换为 `_GCmark`，开启 write barrier，开启 mutator assists，并且入队根集标记任务。只有在所有P都开启写屏障之后才会扫描对象，这通过STW实现。
   2. Start the world。通过mark workers和allocation assist实现GC标记工作。新分配的对象一定是黑色的。
   3. GC执行root marking工作。扫描所有栈、让所有全局变量变灰色、让所有非heap上的运行时数据结构中的堆指针变灰色。扫描栈会暂停一个协程，让栈上的指针变灰色，然后恢复协程执行。
   4. GC将所有的灰色对象的工作队列处理掉，将所有的灰色对象变为黑色，并且将对象中的所有指针标记为灰色（这些指针可能会被加入工作队列）。
   5. 由于GC会遍历所有local cache，因此GC使用一个分布式的停止算法来检测什么时候没有更多的根标记工作或者灰色对象。此时，GC从`_GCmark`阶段到`_GCmarktermination`阶段。
3. 标记结束阶段
   1. stop the world。
   2. 将 `gcphase` 切换为 `_GCmarktermination`，禁止worker和allocation assist。
   3. 做整理操作，比如flush缓存
4. 清扫阶段
   1. 将`gcphase` 切换为 `_GCoff`，设置清扫状态，禁止写屏障
   2. Start the world。新分配的对象是白的。如果有必要，分配会在使用前清扫span。
   3. GC开始后台的并发清扫，并响应分配
5. 当足够多的分配发生时，重新回到阶段1。

并发清扫
1. span-by-span清扫
   1. lazy当协程需要另一个span时再清扫
   2. 并发，在后台协程中一个一个清扫。
2. 在STW mark termination时，所有span被标记为需要清扫
3. 协程需要新span时，先会尝试清扫，回收内存。
   1. 当需要分配新的小对象span时，会不断清扫相同大小的小对象span，直到它清除掉了一个对象
   2. 当需要分配新的大对象span时，会不断清扫span(似乎没有限制)直到它释放了足够多的page
   3. 当协程清扫并释放了两个非相邻的一个页大小的span到heap中，那么还需要继续清扫，因为有可能还有其他一个页大小的未清扫span可能可以合并到这个两个页span中
4. 需要确保没有操作会继续在没有清扫的span上执行，否则可能会导致GC mark bits出错。

`gcController`这个组件是GC步调控制器，这个控制器会决定什么时候触发垃圾收集以及多少mutator assists和后台标记工作要做。

## mcache flush releaseAll

`(*mcache).releaseAll` 会将 `span` 全部归还为 `mcentral`，归还方式为 `(*mcentral).uncacheSpan`，同时还会将 `*mcache` 中的相关参数置零，比如 `tinyAllocs`, `tiny`, `tinyoffset`, `scanAlloc`

mcache flush 会在 `(*mcache).prepareForSweep` 时调用。

## startCycle

清理 `gcController` 的状态。

后台标记协程所占用的P的目标值是所有P数量的25%。即25%的工作用于后台GC标记。

`dedicatedMarkWorkersNeeded`即四舍五入需要的worker数量。

`(*gcControllerState).revise`函数用于调整GC cycle中的assist率

## gcMarkTinyAllocs

将所有p对应的`mcache`中的`tiny`对象

## stopTheWroldWithSema

在停止P方面，首先执行`preemptall`，然后将处于`_Psyscall`的P的状态变为`_Pgcstop`，然后停止idle P，然后等待剩余的P。

`sched.stopwait`用于检测是否所有的P都已暂停。

