#!/bin/bash
# clear malloc_test.log
> profile_result/malloc_test/malloc_test.log

go test -a -v . -bench='^BenchmarkMalloc.*$' -count 10 -benchtime 3s -run='^$' -cpuprofile profile_result/malloc_test/default.out 2>&1 | tee -a profile_result/malloc_test/malloc_test.log > profile_result/malloc_test/default.txt
# tags=(nops_mask nodmb eager_calc)
# for tag in "${tags[@]}"
# do
#     go test -a -v -tags=$tag  . -bench='^BenchmarkMalloc.*$' -count 10 -benchtime 3s -run='^$' -cpuprofile profile_result/malloc_test/$tag.out
# done
go test -a -v -tags=batch . -bench='^BenchmarkMalloc.*$' -count 10 -benchtime 3s -run='^$' -cpuprofile profile_result/malloc_test/batch.out 2>&1 | tee -a profile_result/malloc_test/malloc_test.log > profile_result/malloc_test/batch.txt

cd profile_result/malloc_test
benchstat default.txt batch.txt > benchstat.txt
