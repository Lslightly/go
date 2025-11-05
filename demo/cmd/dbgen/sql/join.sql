-- select after join

USE demo_closure_go;

SELECT package_path, batch_number, node_id, node_location, root_cause_count, root_cause_id
FROM node_info
JOIN node_root_causes
ON node_info.uid = node_root_causes.uid
