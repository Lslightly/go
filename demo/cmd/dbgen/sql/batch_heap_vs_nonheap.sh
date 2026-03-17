#!/bin/bash

# This script queries the heap vs nonheap statistics for a list of databases
# and formats the output into a markdown table.

# List of databases to query.
# Comment out any databases you want to skip.
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

# Get the directory of the script
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
sql_file="$script_dir/heap_vs_nonheap.sql"

for db in "${databases[@]}"; do
    echo "### ${db}"
    echo
    echo "| Heap Count | Non-Heap Count |"
    echo "|------------|----------------|"
    
    clickhouse client --port 19001 --queries-file "$sql_file" --database "${db}" | \
    awk '{printf "| %-10s | %-14s |\n", $1, $2}'
    
    echo
done


# 临时文件保存所有结果
tmp_result=$(mktemp)

# 收集所有数据库结果
for db in "${databases[@]}"; do
    clickhouse client --port 19001 --queries-file "$sql_file" --database "${db}" | awk -v db="$db" '{print db, $1, $2}' >> "$tmp_result"
done

# 输出计数表（数据库为行，Heap Count和Non-Heap Count为列）
echo "## Count Table"
echo
printf "| Database           | Heap Count   | Non-Heap Count |\n"
printf "|--------------------|--------------|----------------|\n"

for db in "${databases[@]}"; do
    heap_val=$(awk -v db="$db" '$1==db {print $2}' "$tmp_result")
    nonheap_val=$(awk -v db="$db" '$1==db {print $3}' "$tmp_result")
    printf "| %-18s | %-12s | %-14s |\n" "$db" "${heap_val:-0}" "${nonheap_val:-0}"
done

rm "$tmp_result"