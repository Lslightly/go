-- 带有ERMust和ERHeap的结点中，除了ERMust和ERHeap原因之外的其他逃逸原因的结点大小分布
-- 以及只包含ERMust和ERHeap，不包含其他逃逸原因的结点大小累计和(标记为ERMust-only)
-- 分母是带有ERMust和ERHeap的结点大小总和

WITH 
-- 获取同时带有ERMust和ERHeap的节点ID及其size
ermust_heap_nodes AS (
    SELECT 
        ni.uid,
        ni.size
    FROM node_info ni
    WHERE EXISTS (
        SELECT 1 FROM node_root_causes nrc1 WHERE ni.uid = nrc1.uid AND nrc1.root_cause_id = 'ERMust'
    )
    AND EXISTS (
        SELECT 1 FROM node_root_causes nrc2 WHERE ni.uid = nrc2.uid AND nrc2.root_cause_id = 'ERHeap'
    )
),
-- 统计这些节点中除了ERMust和ERHeap之外的其他根因的size分布
other_causes_size_stats AS (
    SELECT 
        nrc.root_cause_id,
        SUM(hn.size) as size_sum
    FROM node_root_causes nrc
    INNER JOIN ermust_heap_nodes hn ON nrc.uid = hn.uid
    WHERE nrc.root_cause_id NOT IN ('ERMust', 'ERHeap')
    GROUP BY nrc.root_cause_id
),
-- 计算这些节点的总size
ermust_heap_total_size AS (
    SELECT SUM(size) as total_size FROM ermust_heap_nodes
),
-- 计算只包含ERMust和ERHeap，不包含其他逃逸原因的节点size累加和
ermust_only_size AS (
    SELECT SUM(size) as size_sum
    FROM ermust_heap_nodes hn
    WHERE NOT EXISTS (
        SELECT 1 FROM node_root_causes nrc 
        WHERE hn.uid = nrc.uid 
        AND nrc.root_cause_id NOT IN ('ERMust', 'ERHeap')
    )
)
-- 合并结果：其他根因的size分布 + ERMust-only的size
SELECT 
    ocss.root_cause_id,
    ocss.size_sum,
    ocss.size_sum * 100.0 / ehts.total_size as percentage
FROM other_causes_size_stats ocss
CROSS JOIN ermust_heap_total_size ehts

UNION ALL

-- 添加ERMust-only的数据
SELECT 
    'ERMust-only' as root_cause_id,
    eos.size_sum,
    eos.size_sum * 100.0 / ehts.total_size as percentage
FROM ermust_only_size eos
CROSS JOIN ermust_heap_total_size ehts

ORDER BY size_sum DESC