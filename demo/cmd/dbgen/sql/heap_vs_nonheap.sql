-- 统计带ERHeap的结点数量和不带ERHeap的结点数量
-- 返回两列：heap_count（带ERHeap的结点数量）和 non_heap_count（不带ERHeap的结点数量）

SELECT 
    (SELECT COUNT(DISTINCT uid)
     FROM node_info ni
     JOIN node_root_causes nrc
     ON ni.uid = nrc.uid
     WHERE nrc.root_cause_id = 'ERHeap') AS heap_count,
     
    (SELECT COUNT(DISTINCT ni.uid)
     FROM node_info ni
     WHERE NOT EXISTS (
         SELECT 1
         FROM node_root_causes nrc
         WHERE ni.uid = nrc.uid AND nrc.root_cause_id = 'ERHeap'
     )) AS non_heap_count