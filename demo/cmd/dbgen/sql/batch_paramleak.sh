#!/bin/bash

# This script queries the paramleak statistics for a list of databases
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
    "bench_tidb_server_check"
    "bench_tidb_br_unit_test"
    "bench_tidb_parser_unit_test"
    "bench_tidb_tools_bin_ut"
    "bench_tidb_tools_bin_xprog"
    "bench_tidb_tools_bin_failpoint_ctl"
    "bench_tidb_server"
    "bench_tidb_benchkv"
    "bench_sweet_bleve_index"
    "bench_sweet_cockroachdb"
    "bench_sweet_etcd"
    "bench_sweet_gopher_lua"
    "bench_sweet_hugo"
    "bench_sweet_markdown"
    "bench_sweet_tile38"
)

# Get the directory of the script
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
sql_file="$script_dir/paramleak.sql"

for db in "${databases[@]}"; do
    echo "### ${db}"
    echo
    echo "|       Root Cause     | Count | Percentage |"
    echo "|----------------------|-------|------------|"
    
    clickhouse client --queries-file "$sql_file" --database "${db}" | \
    awk '{printf "| %-20s | %-5s | %.2f%% |\n", $1, $2, $3}'
    
    echo
done