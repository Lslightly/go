#!/bin/bash -x
GOROOT_BOOTSTRAP=/home/lqw/go1.24.0 GO_GCFLAGS="-N -l" ./make.bash
../bin/go env -w GO111MODULE=on
../bin/go env -w GOPROXY=https://goproxy.cn,direct
../bin/go install golang.org/x/tools/gopls@latest
