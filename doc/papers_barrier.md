# 同步屏障相关论文

- [static_code_analyzer_hpcc2022.pdf](https://qcjiang.github.io/pdf/static_code_analyzer_hpcc2022.pdf) 基于鲲鹏 aarch64 的静态分析器，预测 basic block 的吞吐量。
- [No Barrier in the Road: A Comprehensive Study and Optimization of ARM Barriers](https://ipads.se.sjtu.edu.cn/_media/publications/liuppopp20.pdf) **没有比这个更相关的**
- [https://sites.cc.gatech.edu/classes/AY2020/cs8803fmm_fall](https://sites.cc.gatech.edu/classes/AY2020/cs8803mm_spring/) memory model 课程，包含很多经典论文
- Robin Morisset and Francesco Zappa Nardelli. 2017. Partially Redundant Fence Elimination for X86, ARM, and Power Processors.
- Christopher Pulte, Shaked Flur, Will Deacon, Jon French, Susmit Sarkar, and Peter Sewell. 2017. Simplifying ARM Concurrency: Multicopy Atomic Axiomatic and Operational Models for ARMv8. 
- Cycles to Recycle: Garbage Collection on the IA-64 ISMM00

- ASPLOS23 [AtoMig: Automatically Migrating Millions Lines of Code from TSO to WMM](https://dl.acm.org/doi/pdf/10.1145/3575693.3579849)
- Software Language Engineering24 [Reducing Write Barrier Overheads for Orthogonal Persistence](https://dl.acm.org/doi/10.1145/3687997.3695646)
	- 通过逃逸分析发现不被其他线程引用的对象，在这些对象写入时，移除 barrier，减少同步操作，降低屏障开销。
- PPoPP22 [Asymmetry-aware Scalable Locking](https://dl.acm.org/doi/10.1145/3503221.3508420)

关系可能不是很大
- [Optimizing Memory Allocation for Multi-Subgraph Mapping on Spatial Accelerators | Proceedings of the 16th ACM International Conference on Systems and Storage](https://dl.acm.org/doi/abs/10.1145/3579370.3594767) 
- [Machine-aware memory allocation and synchronization](https://www.research-collection.ethz.ch/server/api/core/bitstreams/68748c31-59a3-4eca-aa95-b591d6c579cf/content) 主要是使用 DMA 引擎做优化，和 NUMA 相关，和同步原语关系不是很大。
- [[../../../ZoteroNotes/roofline model|roofline model]] roofline 模型，多核架构优化的关键因素
Garbage Collector-Agnostic Store Barriers for Ahead-of-Time Compilation-硕士论文
Improving Communication by Optimizing On-Node Data Movement with Data Layout
PPoPP25 Effectively Virtual Page Prefetching via Spatial-Temporal Patterns for Memory-intensive Cloud Applications 通过时间-空间模式来提升内存的预取策略。
PPoPP24 Gallatin: A General-Purpose GPU Memory Manager GPU 内存管理器，wait-free vEB tree 来利用 GPU 并行性
无
PPoPP23 Merchandiser: Data Placement on Heterogeneous Memory for Task-Parallel HPC Applications with Load-Balance Awareness 数据放置策略
ASPLOS24 Cornucopia Reloaded: Load Barriers for CHERI Heap Temporal Safety
ASPLOS24 LazyBarrier: Reconstructing Android IO Stack for Barrier-Enabled Flash Storage

