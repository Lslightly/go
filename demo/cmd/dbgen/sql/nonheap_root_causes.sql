-- 查看不带ERHeap的结点包含的其他根因
-- 统计不带ERHeap的结点中，各个根因的出现次数

SELECT 
    nrc.root_cause_id,
    COUNT(DISTINCT nrc.uid) as node_count,
    COUNT(DISTINCT nrc.uid) * 100.0 / SUM(COUNT(DISTINCT nrc.uid)) OVER() as percentage
FROM node_root_causes nrc
WHERE nrc.uid IN (
    -- 找到所有不带 ERHeap 的节点
    SELECT DISTINCT ni.uid 
    FROM node_info ni
    WHERE NOT EXISTS (
        SELECT 1
        FROM node_root_causes nrc2
        WHERE ni.uid = nrc2.uid AND nrc2.root_cause_id = 'ERHeap'
    )
)
GROUP BY nrc.root_cause_id
ORDER BY node_count DESC