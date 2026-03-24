-- 查看带ERHeap的结点包含的其他根因（排除ERHeap本身）
-- 统计带ERHeap的结点中，各个其他根因对应的节点size累加和
-- 百分比基于带ERHeap的节点的size累加和总量

WITH 
-- 获取所有带ERHeap的节点ID及其size
heap_nodes AS (
    SELECT 
        ni.uid,
        ni.size
    FROM node_info ni
    WHERE EXISTS (
        SELECT 1
        FROM node_root_causes nrc
        WHERE ni.uid = nrc.uid AND nrc.root_cause_id = 'ERHeap'
    )
),
-- 获取带ERHeap的节点中，除了ERHeap之外的其他根因的size统计
cause_size_stats AS (
    SELECT 
        nrc.root_cause_id,
        SUM(hn.size) as size_sum
    FROM node_root_causes nrc
    INNER JOIN heap_nodes hn ON nrc.uid = hn.uid
    WHERE nrc.root_cause_id != 'ERHeap'
    GROUP BY nrc.root_cause_id
)
SELECT 
    css.root_cause_id,
    css.size_sum,
    css.size_sum * 100.0 / (SELECT SUM(size) FROM heap_nodes) as percentage
FROM cause_size_stats css
ORDER BY css.size_sum DESC