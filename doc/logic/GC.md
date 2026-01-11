# GO GC

GC bitmap gcmask用于标记对象内每8字节内存是否包含指针

gcmarkBits用于确定当前对象是否已被扫描。0表示白色，1表示黑色。如果对象在工作队列中，则表示灰色，等待扫描子对象。

## goref

[goref design priciple](https://github.com/cloudwego/goref/blob/main/docs/principle.md) 设计原则，其中包含了对GC的部分描述。

