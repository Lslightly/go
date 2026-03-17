#!/bin/bash

# =============================================================================
# 批量查询非堆根因统计脚本
# 
# 功能：
# 1. 查询所有数据库中不带ERHeap的结点所包含的逃逸根因统计
# 2. 生成Markdown表格输出到控制台（每个数据库的详细统计）
# 3. 生成CSV文件（数据库为行、根因为列的count和percentage数据）
# 4. 生成CSV文件（根因为行、数据库为列的count和percentage数据，转置版本）
# 5. 调用Python脚本生成热力图
#
# 输出文件：
# - nonheap_root_causes_count.csv: Count数据（数据库为行）
# - nonheap_root_causes_percentage.csv: Percentage数据（数据库为行）
# - nonheap_root_causes_count_transpose.csv: Count数据（根因为行，转置版本）
# - nonheap_root_causes_percentage_transpose.csv: Percentage数据（根因为行，转置版本）
# - nonheap_root_causes_count_heatmap.png: count热力图
# - nonheap_root_causes_percentage_heatmap.png: percentage热力图
# =============================================================================

# =============================================================================
# 配置部分
# =============================================================================

# 数据库列表 - 在此处配置需要查询的数据库
# 注释掉不需要查询的数据库
databases=(
    "bench_cli"
    "bench_fzf"
    "bench_gogs"
    "bench_grpc_go"
    "bench_hugo"
    "bench_rclone"
    # "bench_tidb_server_check"
    # "bench_tidb_br_unit_test"
    # "bench_tidb_parser_unit_test"
    # "bench_tidb_tools_bin_ut"
    # "bench_tidb_tools_bin_xprog"
    # "bench_tidb_tools_bin_failpoint_ctl"
    # "bench_tidb_server"
    # "bench_tidb_benchkv"
    # "bench_sweet_biogo_krishna"
    # "bench_sweet_bleve_index"
    # "bench_sweet_cockroachdb"
    # "bench_sweet_esbuild"
    # "bench_sweet_etcd"
    # "bench_sweet_go_build"
    # "bench_sweet_gopher_lua"
    # "bench_sweet_hugo"
    # "bench_sweet_markdown"
    # "bench_sweet_tile38"
)

# ClickHouse连接配置
CLICKHOUSE_PORT=19001

# 数据库到仓库名称的映射表
# 格式: 数据库名:仓库名
declare -A db_repo_mapping=(
    ["bench_cli"]="cli"
    ["bench_fzf"]="fzf"
    ["bench_gogs"]="gogs"
    ["bench_grpc_go"]="grpc-go"
    ["bench_hugo"]="hugo"
    ["bench_rclone"]="rclone"
)

# 获取脚本所在目录
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
output_dir=$script_dir/output

# 创建输出目录
mkdir -p "$output_dir"

# SQL文件路径
sql_file="$script_dir/nonheap_root_causes.sql"

# 输出文件路径
csv_count_file="$output_dir/nonheap_root_causes_count.csv"
csv_percentage_file="$output_dir/nonheap_root_causes_percentage.csv"
csv_count_transpose_file="$output_dir/nonheap_root_causes_count_transpose.csv"
csv_percentage_transpose_file="$output_dir/nonheap_root_causes_percentage_transpose.csv"

# 获取仓库名称的函数
get_repo_name() {
    local db_name=$1
    local repo_name="${db_repo_mapping[$db_name]}"
    if [ -z "$repo_name" ]; then
        # 如果映射表中没有，移除bench_前缀
        repo_name="${db_name#bench_}"
    fi
    echo "$repo_name"
}

# 获取所有仓库名称列表（用于CSV表头）
repo_names=()
for db in "${databases[@]}"; do
    repo_names+=("$(get_repo_name "$db")")
done

# 临时文件用于收集所有数据库的原始数据
tmp_raw_data=$(mktemp)

# =============================================================================
# 第一部分：查询所有数据库并收集原始数据
# =============================================================================

echo "开始查询数据库..."
echo

# 遍历所有数据库，执行查询并将结果收集到临时文件中
for db in "${databases[@]}"; do
    echo "查询数据库: ${db}"
    
    # 执行SQL查询，将结果格式化为：数据库,根因,结点数量,百分比
    clickhouse client --port "$CLICKHOUSE_PORT" --queries-file "$sql_file" --database "${db}" | \
    awk -v db="$db" '{print db "," $1 "," $2 "," $3}' >> "$tmp_raw_data"
done

echo
echo "查询完成！"
echo

# =============================================================================
# 第二部分：输出每个数据库的详细统计（Markdown格式）
# =============================================================================

echo "=========================================================================="
echo "每个数据库的非堆根因统计（Markdown格式）"
echo "=========================================================================="
echo

# 临时文件用于保存当前数据库的查询结果
tmp_db_result=$(mktemp)

for db in "${databases[@]}"; do
    # 从原始数据中提取当前数据库的结果
    awk -F',' -v db="$db" '$1==db {print $0}' "$tmp_raw_data" > "$tmp_db_result"
    
    # 检查是否有数据
    if [ -s "$tmp_db_result" ]; then
        repo_name=$(get_repo_name "$db")
        echo "### ${repo_name}"
        echo
        echo "| 根因                | 数量   | 百分比     |"
        echo "|---------------------|--------|------------|"
        
        # 输出表格行
        while IFS=',' read -r db_name root_cause count percentage; do
            printf "| %-19s | %-6s | %.2f%% |\n" "$root_cause" "$count" "$percentage"
        done < "$tmp_db_result"
        
        echo
    fi
done

# 清理临时文件
rm "$tmp_db_result"

# =============================================================================
# 第三部分：收集所有根因类型（动态收集，不硬编码）
# =============================================================================

# 从原始数据中提取所有唯一的根因类型并排序
root_causes=($(awk -F',' '{print $2}' "$tmp_raw_data" | sort | uniq))

echo "发现的根因类型: ${root_causes[*]}"
echo

# =============================================================================
# 第四部分：生成CSV文件 - 数据库为行，根因为列（中间目标）
# =============================================================================

# 生成Count表格
echo "生成CSV文件（Count - 仓库为行）..."
printf "仓库" > "$csv_count_file"
for cause in "${root_causes[@]}"; do
    printf ",%s" "$cause" >> "$csv_count_file"
done
echo >> "$csv_count_file"

for db in "${databases[@]}"; do
    repo_name=$(get_repo_name "$db")
    printf "%s" "$repo_name" >> "$csv_count_file"
    for cause in "${root_causes[@]}"; do
        # 从原始数据中查找对应的count值
        val=$(awk -F',' -v db="$db" -v cause="$cause" '$1==db && $2==cause {print $3}' "$tmp_raw_data")
        printf ",%s" "${val:-0}" >> "$csv_count_file"
    done
    echo >> "$csv_count_file"
done

# 生成Percentage表格
echo "生成CSV文件（Percentage - 仓库为行）..."
printf "仓库" > "$csv_percentage_file"
for cause in "${root_causes[@]}"; do
    printf ",%s" "$cause" >> "$csv_percentage_file"
done
echo >> "$csv_percentage_file"

for db in "${databases[@]}"; do
    repo_name=$(get_repo_name "$db")
    printf "%s" "$repo_name" >> "$csv_percentage_file"
    for cause in "${root_causes[@]}"; do
        # 从原始数据中查找对应的percentage值
        val=$(awk -F',' -v db="$db" -v cause="$cause" '$1==db && $2==cause {printf "%.2f", $4}' "$tmp_raw_data")
        printf ",%s%%" "${val:-0}" >> "$csv_percentage_file"
    done
    echo >> "$csv_percentage_file"
done

# =============================================================================
# 第五部分：生成CSV文件 - 根因为行，数据库为列（最终目标，转置版本）
# =============================================================================

# 生成Count表格（转置）到单独文件
echo "生成CSV文件（Count - 根因为行，转置版本）..."
printf "根因" > "$csv_count_transpose_file"
for db in "${databases[@]}"; do
    repo_name=$(get_repo_name "$db")
    printf ",%s" "$repo_name" >> "$csv_count_transpose_file"
done
echo >> "$csv_count_transpose_file"

for cause in "${root_causes[@]}"; do
    printf "%s" "$cause" >> "$csv_count_transpose_file"
    for db in "${databases[@]}"; do
        # 从原始数据中查找对应的count值
        val=$(awk -F',' -v db="$db" -v cause="$cause" '$1==db && $2==cause {print $3}' "$tmp_raw_data")
        printf ",%s" "${val:-0}" >> "$csv_count_transpose_file"
    done
    echo >> "$csv_count_transpose_file"
done

# 生成Percentage表格（转置）到单独文件
echo "生成CSV文件（Percentage - 根因为行，转置版本）..."
printf "根因" > "$csv_percentage_transpose_file"
for db in "${databases[@]}"; do
    repo_name=$(get_repo_name "$db")
    printf ",%s" "$repo_name" >> "$csv_percentage_transpose_file"
done
echo >> "$csv_percentage_transpose_file"

for cause in "${root_causes[@]}"; do
    printf "%s" "$cause" >> "$csv_percentage_transpose_file"
    for db in "${databases[@]}"; do
        # 从原始数据中查找对应的percentage值
        val=$(awk -F',' -v db="$db" -v cause="$cause" '$1==db && $2==cause {printf "%.2f", $4}' "$tmp_raw_data")
        printf ",%s%%" "${val:-0}" >> "$csv_percentage_transpose_file"
    done
    echo >> "$csv_percentage_transpose_file"
done

echo "CSV文件生成完成："
echo "  - $csv_count_file"
echo "  - $csv_percentage_file"
echo "  - $csv_count_transpose_file"
echo "  - $csv_percentage_transpose_file"
echo

# =============================================================================
# 第六部分：调用Python脚本生成热力图
# =============================================================================

echo "生成热力图..."
python3 "$script_dir/generate_heatmap.py" \
    --count-file "$csv_count_transpose_file" \
    --percentage-file "$csv_percentage_transpose_file" \
    --output-dir "$output_dir"

echo "热力图生成完成！"
echo

# =============================================================================
# 清理临时文件
# =============================================================================

rm "$tmp_raw_data"

echo "=========================================================================="
echo "所有任务完成！"
echo "=========================================================================="
echo
echo "生成的文件："
echo "1. Markdown表格 - 已输出到控制台"
echo "2. $csv_count_file - Count数据（仓库为行）"
echo "3. $csv_percentage_file - Percentage数据（仓库为行）"
echo "4. $csv_count_transpose_file - Count数据（根因为行，转置版本）"
echo "5. $csv_percentage_transpose_file - Percentage数据（根因为行，转置版本）"
echo "6. output/nonheap_root_causes_count_heatmap.png - Count热力图"
echo "7. output/nonheap_root_causes_percentage_heatmap.png - Percentage热力图"
echo