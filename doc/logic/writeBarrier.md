# writeBarrier开启条件

如果不考虑 hall of shame，`writeBarrier.enabled` 的更新点只有 `setGCPhase`。`setGCPhase` 的调用包括：
1. `gcStart` 调用 `setGCPhase(_GCmark)`
2. `gcStart` 调用 `gcMarkTermination` 调用 `setGCPhase(_GCmarktermination)`
3. `gcStart` 调用 `gcMarkTermination` 调用 `setGCPhase(_GCoff)`


