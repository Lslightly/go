#!/bin/bash
go test -a . -bench='^BenchmarkMalloc.*$' -count 10 -benchtime 5s -run='^$' -cpuprofile profile_result/malloc_test/default.out
tags=(nops_mask nodmb eager_calc)
for tag in "${tags[@]}"
do
    go test -a -tags=$tag  . -bench='^BenchmarkMalloc.*$' -count 10 -benchtime 5s -run='^$' -cpuprofile profile_result/malloc_test/$tag.out
done
go test -a -tags=batch . -bench='^BenchmarkMalloc.*$' -count 10 -benchtime 5s -run='^$' -cpuprofile profile_result/malloc_test/batch.out
