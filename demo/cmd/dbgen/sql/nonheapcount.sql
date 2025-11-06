-- 统计不带ERHeap的结点数量

USE bench_hugo;

SELECT COUNT(DISTINCT ni.uid)
FROM node_info ni
WHERE
    NOT EXISTS (
        SELECT 1
        FROM node_root_causes nrc
        WHERE ni.uid = nrc.uid AND nrc.root_cause_id = 'ERHeap'
    )
