#!/bin/bash
go test -a -v . -bench='^BenchmarkMalloc.*$' -run='^$' -trace profile_result/malloc_test/default.trace.out
go test -a -v -tags=batch . -bench='^BenchmarkMalloc.*$' -run='^$' -trace profile_result/malloc_test/batch.trace.out

