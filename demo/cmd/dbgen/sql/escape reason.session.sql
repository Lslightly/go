SELECT
    'escapeReasonStatRoot/fzf/graphs/' || ni.package_path || '/' || CAST(ni.batch_number AS String) || '/summarize.gv' AS dir,
    ni.node_id,
    ni.node_location,
    nrc.root_cause_id
FROM bench_fzf.node_info ni
JOIN bench_fzf.node_root_causes nrc
ON ni.uid = nrc.uid
WHERE nrc.root_cause_id = 'ERGo' and ni.node_location not like '/home/lqw/go/pkg/mod/%'
ORDER BY rand()
LIMIT 20
;
