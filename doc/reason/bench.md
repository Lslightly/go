# benchmark逃逸根因统计结果

[paramleak.sql](../../demo/cmd/dbgen/sql/paramleak.sql)用于统计除掉 `ERParamLeak` 和 `ERHeap` 之外的各个逃逸根因的出现数量占比。

使用[paramleak.sql](../../demo/cmd/dbgen/sql/paramleak.sql)查询得到的分类结果如下。

### bench_cli

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 33901 | 57.67% |
| ERStar               | 8772  | 14.92% |
| ERMapIdx             | 6849  | 11.65% |
| ERClosure            | 3924  | 6.67% |
| ERAppend             | 2166  | 3.68% |
| ERSliceIdx           | 1305  | 2.22% |
| ERGo                 | 907   | 1.54% |
| ERCopy               | 333   | 0.57% |
| EROutliveRootStack   | 246   | 0.42% |
| ERComm               | 184   | 0.31% |
| ERMust               | 105   | 0.18% |
| ERGlobal             | 54    | 0.09% |
| ERDefer              | 42    | 0.07% |

### bench_fzf

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 267   | 25.55% |
| ERClosure            | 196   | 18.76% |
| ERStar               | 182   | 17.42% |
| ERGo                 | 119   | 11.39% |
| ERMapIdx             | 84    | 8.04% |
| ERAppend             | 73    | 6.99% |
| ERComm               | 54    | 5.17% |
| EROutliveRootStack   | 23    | 2.20% |
| ERCopy               | 23    | 2.20% |
| ERSliceIdx           | 14    | 1.34% |
| ERMust               | 9     | 0.86% |
| ERGlobal             | 1     | 0.10% |

### bench_gogs

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 19152 | 48.54% |
| ERStar               | 8109  | 20.55% |
| ERAppend             | 5998  | 15.20% |
| ERClosure            | 2594  | 6.57% |
| ERMapIdx             | 1606  | 4.07% |
| ERSliceIdx           | 914   | 2.32% |
| EROutliveRootStack   | 328   | 0.83% |
| ERGo                 | 251   | 0.64% |
| ERMust               | 183   | 0.46% |
| ERCopy               | 140   | 0.35% |
| ERComm               | 86    | 0.22% |
| ERGlobal             | 47    | 0.12% |
| ERDefer              | 46    | 0.12% |

### bench_grpc_go

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 22951 | 62.27% |
| ERStar               | 4574  | 12.41% |
| ERClosure            | 3193  | 8.66% |
| ERAppend             | 1786  | 4.85% |
| ERMapIdx             | 1517  | 4.12% |
| ERSliceIdx           | 1316  | 3.57% |
| ERGo                 | 593   | 1.61% |
| ERComm               | 479   | 1.30% |
| ERCopy               | 269   | 0.73% |
| EROutliveRootStack   | 101   | 0.27% |
| ERMust               | 47    | 0.13% |
| ERGlobal             | 22    | 0.06% |
| ERDefer              | 12    | 0.03% |

### bench_hugo

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 27586 | 39.21% |
| ERStar               | 16161 | 22.97% |
| ERClosure            | 9055  | 12.87% |
| ERMapIdx             | 7961  | 11.32% |
| ERGo                 | 3249  | 4.62% |
| ERSliceIdx           | 2868  | 4.08% |
| ERAppend             | 1910  | 2.72% |
| ERCopy               | 646   | 0.92% |
| EROutliveRootStack   | 400   | 0.57% |
| ERComm               | 284   | 0.40% |
| ERMust               | 92    | 0.13% |
| ERDefer              | 88    | 0.13% |
| ERGlobal             | 48    | 0.07% |

### bench_rclone

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 37832 | 45.78% |
| ERStar               | 14608 | 17.68% |
| ERClosure            | 12756 | 15.44% |
| ERMapIdx             | 7499  | 9.07% |
| ERAppend             | 3583  | 4.34% |
| ERGo                 | 3053  | 3.69% |
| ERSliceIdx           | 1593  | 1.93% |
| ERCopy               | 628   | 0.76% |
| ERComm               | 480   | 0.58% |
| ERMust               | 255   | 0.31% |
| EROutliveRootStack   | 234   | 0.28% |
| ERGlobal             | 62    | 0.08% |
| ERDefer              | 51    | 0.06% |

### bench_tidb_server_check

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 111714 | 49.67% |
| ERStar               | 58219 | 25.88% |
| ERSliceIdx           | 14310 | 6.36% |
| ERClosure            | 14144 | 6.29% |
| ERAppend             | 6786  | 3.02% |
| ERMapIdx             | 6361  | 2.83% |
| ERCopy               | 4917  | 2.19% |
| ERGo                 | 4803  | 2.14% |
| ERComm               | 1541  | 0.69% |
| ERMust               | 1047  | 0.47% |
| EROutliveRootStack   | 758   | 0.34% |
| ERGlobal             | 247   | 0.11% |
| ERDefer              | 70    | 0.03% |

### bench_tidb_br_unit_test

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 125987 | 48.90% |
| ERStar               | 65429 | 25.40% |
| ERClosure            | 16923 | 6.57% |
| ERSliceIdx           | 16417 | 6.37% |
| ERAppend             | 8349  | 3.24% |
| ERMapIdx             | 7227  | 2.81% |
| ERCopy               | 7040  | 2.73% |
| ERGo                 | 6060  | 2.35% |
| ERComm               | 1751  | 0.68% |
| ERMust               | 1217  | 0.47% |
| EROutliveRootStack   | 790   | 0.31% |
| ERGlobal             | 355   | 0.14% |
| ERDefer              | 71    | 0.03% |

### bench_tidb_parser_unit_test

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 2319  | 47.59% |
| ERStar               | 1195  | 24.52% |
| ERAppend             | 837   | 17.18% |
| ERSliceIdx           | 155   | 3.18% |
| ERClosure            | 145   | 2.98% |
| ERMapIdx             | 96    | 1.97% |
| ERCopy               | 54    | 1.11% |
| ERGo                 | 31    | 0.64% |
| ERMust               | 24    | 0.49% |
| EROutliveRootStack   | 12    | 0.25% |
| ERGlobal             | 4     | 0.08% |
| ERComm               | 1     | 0.02% |

### bench_tidb_tools_bin_ut

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 48    | 69.57% |
| ERStar               | 7     | 10.14% |
| ERClosure            | 5     | 7.25% |
| ERMapIdx             | 3     | 4.35% |
| ERAppend             | 3     | 4.35% |
| ERGo                 | 2     | 2.90% |
| EROutliveRootStack   | 1     | 1.45% |

### bench_tidb_tools_bin_xprog

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|

### bench_tidb_tools_bin_failpoint_ctl

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 80    | 56.34% |
| ERAppend             | 25    | 17.61% |
| EROutliveRootStack   | 14    | 9.86% |
| ERStar               | 13    | 9.15% |
| ERCopy               | 7     | 4.93% |
| ERSliceIdx           | 3     | 2.11% |

### bench_tidb_server

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 111610 | 49.67% |
| ERStar               | 58224 | 25.91% |
| ERSliceIdx           | 14310 | 6.37% |
| ERClosure            | 14108 | 6.28% |
| ERAppend             | 6784  | 3.02% |
| ERMapIdx             | 6304  | 2.81% |
| ERCopy               | 4917  | 2.19% |
| ERGo                 | 4779  | 2.13% |
| ERComm               | 1548  | 0.69% |
| ERMust               | 1048  | 0.47% |
| EROutliveRootStack   | 757   | 0.34% |
| ERGlobal             | 249   | 0.11% |
| ERDefer              | 70    | 0.03% |

### bench_tidb_benchkv

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 108119 | 49.80% |
| ERStar               | 56308 | 25.94% |
| ERSliceIdx           | 13794 | 6.35% |
| ERClosure            | 13741 | 6.33% |
| ERAppend             | 6610  | 3.04% |
| ERMapIdx             | 5948  | 2.74% |
| ERGo                 | 4657  | 2.15% |
| ERCopy               | 4389  | 2.02% |
| ERComm               | 1546  | 0.71% |
| ERMust               | 1044  | 0.48% |
| EROutliveRootStack   | 742   | 0.34% |
| ERGlobal             | 124   | 0.06% |
| ERDefer              | 70    | 0.03% |

### bench_sweet_biogo_krishna

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 403   | 48.21% |
| ERStar               | 238   | 28.47% |
| ERAppend             | 66    | 7.89% |
| ERClosure            | 39    | 4.67% |
| ERMapIdx             | 33    | 3.95% |
| ERGo                 | 20    | 2.39% |
| ERCopy               | 12    | 1.44% |
| ERSliceIdx           | 11    | 1.32% |
| ERComm               | 9     | 1.08% |
| EROutliveRootStack   | 3     | 0.36% |
| ERMust               | 2     | 0.24% |

### bench_sweet_bleve_index

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 3413  | 50.38% |
| ERStar               | 1741  | 25.70% |
| ERSliceIdx           | 474   | 7.00% |
| ERClosure            | 387   | 5.71% |
| ERMapIdx             | 256   | 3.78% |
| ERAppend             | 190   | 2.80% |
| ERMust               | 118   | 1.74% |
| ERGo                 | 111   | 1.64% |
| ERComm               | 32    | 0.47% |
| EROutliveRootStack   | 25    | 0.37% |
| ERCopy               | 25    | 0.37% |
| ERGlobal             | 3     | 0.04% |

### bench_sweet_cockroachdb

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 138327 | 50.46% |
| ERStar               | 63501 | 23.17% |
| ERClosure            | 19145 | 6.98% |
| ERCopy               | 16633 | 6.07% |
| ERMapIdx             | 10568 | 3.86% |
| ERSliceIdx           | 9070  | 3.31% |
| ERGo                 | 7161  | 2.61% |
| ERAppend             | 5791  | 2.11% |
| ERMust               | 1324  | 0.48% |
| EROutliveRootStack   | 1143  | 0.42% |
| ERComm               | 820   | 0.30% |
| ERDefer              | 424   | 0.15% |
| ERGlobal             | 217   | 0.08% |

### bench_sweet_esbuild

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 1472  | 38.85% |
| ERStar               | 709   | 18.71% |
| ERClosure            | 333   | 8.79% |
| ERAppend             | 327   | 8.63% |
| EROutliveRootStack   | 265   | 6.99% |
| ERSliceIdx           | 220   | 5.81% |
| ERMapIdx             | 185   | 4.88% |
| ERGo                 | 111   | 2.93% |
| ERDefer              | 78    | 2.06% |
| ERCopy               | 50    | 1.32% |
| ERMust               | 37    | 0.98% |
| ERComm               | 2     | 0.05% |

### bench_sweet_etcd

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 9565  | 46.99% |
| ERStar               | 3836  | 18.84% |
| ERClosure            | 2309  | 11.34% |
| ERMapIdx             | 1676  | 8.23% |
| ERSliceIdx           | 863   | 4.24% |
| ERGo                 | 806   | 3.96% |
| ERAppend             | 683   | 3.36% |
| ERComm               | 246   | 1.21% |
| ERCopy               | 117   | 0.57% |
| ERGlobal             | 92    | 0.45% |
| EROutliveRootStack   | 81    | 0.40% |
| ERMust               | 81    | 0.40% |
| ERDefer              | 1     | 0.00% |

### bench_sweet_go_build

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 239   | 42.30% |
| ERStar               | 113   | 20.00% |
| ERAppend             | 63    | 11.15% |
| ERMapIdx             | 49    | 8.67% |
| ERComm               | 32    | 5.66% |
| ERClosure            | 25    | 4.42% |
| ERCopy               | 21    | 3.72% |
| ERSliceIdx           | 13    | 2.30% |
| EROutliveRootStack   | 5     | 0.88% |
| ERMust               | 3     | 0.53% |
| ERGo                 | 1     | 0.18% |
| ERGlobal             | 1     | 0.18% |

### bench_sweet_gopher_lua

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 378   | 32.70% |
| ERSliceIdx           | 329   | 28.46% |
| ERStar               | 248   | 21.45% |
| ERAppend             | 90    | 7.79% |
| ERMapIdx             | 54    | 4.67% |
| ERClosure            | 30    | 2.60% |
| ERMust               | 10    | 0.87% |
| ERCopy               | 8     | 0.69% |
| EROutliveRootStack   | 6     | 0.52% |
| ERGo                 | 3     | 0.26% |

### bench_sweet_hugo

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 29409 | 41.25% |
| ERStar               | 15522 | 21.77% |
| ERMapIdx             | 9680  | 13.58% |
| ERClosure            | 8291  | 11.63% |
| ERGo                 | 3253  | 4.56% |
| ERSliceIdx           | 2115  | 2.97% |
| ERAppend             | 1820  | 2.55% |
| EROutliveRootStack   | 387   | 0.54% |
| ERCopy               | 330   | 0.46% |
| ERComm               | 283   | 0.40% |
| ERMust               | 91    | 0.13% |
| ERDefer              | 86    | 0.12% |
| ERGlobal             | 19    | 0.03% |

### bench_sweet_markdown

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 164   | 39.90% |
| ERStar               | 104   | 25.30% |
| ERAppend             | 62    | 15.09% |
| ERClosure            | 30    | 7.30% |
| ERMapIdx             | 26    | 6.33% |
| ERSliceIdx           | 12    | 2.92% |
| ERCopy               | 5     | 1.22% |
| ERMust               | 5     | 1.22% |
| EROutliveRootStack   | 2     | 0.49% |
| ERGo                 | 1     | 0.24% |

### bench_sweet_tile38

|       Root Cause     | Count | Percentage |
|----------------------|-------|------------|
| ERReturn             | 6276  | 51.51% |
| ERStar               | 2097  | 17.21% |
| ERClosure            | 2028  | 16.64% |
| ERGo                 | 492   | 4.04% |
| ERMapIdx             | 475   | 3.90% |
| ERAppend             | 335   | 2.75% |
| ERComm               | 186   | 1.53% |
| ERSliceIdx           | 166   | 1.36% |
| EROutliveRootStack   | 52    | 0.43% |
| ERCopy               | 51    | 0.42% |
| ERMust               | 22    | 0.18% |
| ERGlobal             | 4     | 0.03% |



