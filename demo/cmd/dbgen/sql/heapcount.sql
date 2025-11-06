-- 统计带ERHeap的结点数量

USE bench_hugo;

SELECT COUNT(DISTINCT uid)
FROM node_info ni
JOIN node_root_causes nrc
ON ni.uid = nrc.uid
WHERE nrc.root_cause_id = 'ERHeap'
