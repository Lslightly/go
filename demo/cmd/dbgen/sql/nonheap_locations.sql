-- 查看不带ERHeap的结点的node_location
-- 限制返回前100行，可以调整LIMIT的数量

SELECT 
    ni.uid,
    ni.node_location,
    ni.package_path,
    ni.root_cause_count
FROM node_info ni
WHERE NOT EXISTS (
    SELECT 1
    FROM node_root_causes nrc
    WHERE ni.uid = nrc.uid AND nrc.root_cause_id = 'ERHeap'
)
ORDER BY ni.uid
LIMIT 100