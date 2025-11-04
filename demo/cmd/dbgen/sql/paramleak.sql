-- 将带有ERParamLeak的原因计数做进一步占比细分
-- 首先选择带ERParamLeak的结点，然后统计这些结点根因中除了ERParamLeak之外的原因数量

SELECT 
    nrc.root_cause_id,
    nrc.root_cause_description,
    COUNT(*) as count,
    COUNT(*) * 100.0 / SUM(COUNT(*)) OVER() as percentage
FROM node_root_causes nrc
WHERE nrc.uid IN (
    -- 找到所有带有 ERParamLeak (root_cause_id = 256) 的节点
    SELECT DISTINCT uid 
    FROM node_root_causes 
    WHERE root_cause_id = 256
)
AND nrc.root_cause_id != 256  -- 排除 ERParamLeak 本身
GROUP BY nrc.root_cause_id, nrc.root_cause_description
ORDER BY count DESC;
