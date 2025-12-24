#!/bin/bash
go test -a . -bench='^BenchmarkMalloc.*$' -run='none' -cpuprofile profile_result/malloc_test/cpu.out
tags=(nops_mask nodmb eager_calc)
for tag in "${tags[@]}"
do
    go test -a -tags=$tag  . -bench='^BenchmarkMalloc.*$' -run='none' -cpuprofile profile_result/malloc_test/$tag.out
done

