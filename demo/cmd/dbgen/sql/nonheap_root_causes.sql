-- 查看不带ERHeap的结点包含的其他根因
-- 统计不带ERHeap的结点中，各个根因的出现次数
-- 百分比基于不带ERHeap的总节点数

WITH 
-- 获取所有不带ERHeap的节点ID
nonheap_nodes AS (
    SELECT DISTINCT uid
    FROM node_info ni
    WHERE NOT EXISTS (
        SELECT 1
        FROM node_root_causes nrc
        WHERE ni.uid = nrc.uid AND nrc.root_cause_id = 'ERHeap'
    )
),
-- 获取不带ERHeap的节点中的根因统计
cause_stats AS (
    SELECT 
        nrc.root_cause_id,
        COUNT(DISTINCT nrc.uid) as node_count
    FROM node_root_causes nrc
    WHERE nrc.uid IN (SELECT uid FROM nonheap_nodes)
    GROUP BY nrc.root_cause_id
)
SELECT 
    cs.root_cause_id,
    cs.node_count,
    cs.node_count * 100.0 / (SELECT COUNT(*) FROM nonheap_nodes) as percentage
FROM cause_stats cs
ORDER BY cs.node_count DESC