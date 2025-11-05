-- 查询带x根因的结点的信息
USE bench_fzf;

SELECT *
FROM node_info ni
WHERE ni.uid IN (
    -- 找到所有带有根因x的结点
    SELECT DISTINCT uid
    FROM node_root_causes
    WHERE root_cause_id = 'ERHeap' -- ERHeap
)
ORDER BY uid
