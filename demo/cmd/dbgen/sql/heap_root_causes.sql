-- 查看带ERHeap的结点包含的其他根因（排除ERHeap本身）
-- 统计带ERHeap的结点中，各个其他根因的出现次数
-- 百分比基于带ERHeap的总节点数（排除ERHeap本身的根因统计）

WITH 
-- 获取所有带ERHeap的节点ID（不包含ERHeap根因）
heap_nodes AS (
    SELECT DISTINCT uid
    FROM node_root_causes
    WHERE root_cause_id = 'ERHeap'
),
-- 获取带ERHeap的节点中，除了ERHeap之外的其他根因统计
cause_stats AS (
    SELECT 
        nrc.root_cause_id,
        COUNT(DISTINCT nrc.uid) as node_count
    FROM node_root_causes nrc
    WHERE nrc.uid IN (SELECT uid FROM heap_nodes)
    AND nrc.root_cause_id != 'ERHeap'
    GROUP BY nrc.root_cause_id
)
SELECT 
    cs.root_cause_id,
    cs.node_count,
    cs.node_count * 100.0 / (SELECT COUNT(*) FROM heap_nodes) as percentage
FROM cause_stats cs
ORDER BY cs.node_count DESC