-- 计算包含 ERGo 的节点数量，相对于 ERHeap 节点和总节点的占比
SELECT
    -- ERGo 节点数量
    uniqIf(ni.uid, nrc.root_cause_id = 'ERGo') AS er_go_count,

    -- ERHeap 节点数量
    uniqIf(ni.uid, nrc.root_cause_id = 'ERHeap') AS er_heap_count,

    -- 所有节点数量
    COUNT(DISTINCT ni.uid) AS total_node_count,

    -- ERGo / ERHeap 百分比（避免除零）
    round(
        if(er_heap_count = 0, 0, (er_go_count * 100.0) / er_heap_count),
        2
    ) AS percent_of_erheap,

    -- ERGo / 总节点 百分比
    round(
        (er_go_count * 100.0) / total_node_count,
        2
    ) AS percent_of_total
FROM node_info ni
LEFT JOIN node_root_causes nrc ON ni.uid = nrc.uid;
