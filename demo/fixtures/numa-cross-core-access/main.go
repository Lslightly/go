package main

/*
#define _GNU_SOURCE
#include <sched.h>
*/
import "C"

import (
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

const (
	N       int = 1_000_000_000 // 访问次数
	fixLen  int = 512 * 1024 * 1024 / 8
	CoreNum int = 256
)

func getCurrentCPU() int {
	return int(C.sched_getcpu())
}

func main() {
	runtime.GOMAXPROCS(CoreNum)
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	localTime := accessLocalAlloc()
	runtime.GC()
	println("-------\nlocal\n------------------\nremote\n-------")
	remoteTime := accessRemoteAlloc()
	println("remote alloc access time:", remoteTime.String())
	println("local alloc access time:", localTime.String())
	println("remote/local:", float64(remoteTime)/float64(localTime))
	{
		f, err := os.Create("mem.prof")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

func averageTime(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func accessRemoteAlloc() time.Duration {
	timeCh := make(chan time.Duration, CoreNum/2)
	useSumCh := make(chan int, CoreNum/2)
	set := genSet()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for i := range CoreNum / 2 {
		if inSet(i, set) {
			println(i, "alloc at core:", getCurrentCPU())
		}
		remoteData := make([]int, fixLen) // 512M
		go func(i int) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			if inSet(i, set) {
				println(i, "local core begin:", getCurrentCPU())
			}
			start := time.Now()
			sum := 0
			for j := range N {
				remoteData[j%fixLen] = j
				sum += remoteData[j%fixLen]
			}
			elapsed := time.Since(start)
			timeCh <- elapsed
			useSumCh <- sum
			if inSet(i, set) {
				println(i, "local core finish:", getCurrentCPU())
			}
		}(i)
	}
	durations := make([]time.Duration, CoreNum/2)
	for i := range CoreNum / 2 {
		durations[i] = <-timeCh
	}
	for range CoreNum / 2 {
		<-useSumCh
	}
	return averageTime(durations)
}

func genSet() map[int]struct{} {
	set := make(map[int]struct{})
	for i := 2; i < 5; i++ {
		set[i] = struct{}{}
	}
	return set
}

func inSet(i int, set map[int]struct{}) bool {
	_, ok := set[i]
	return ok
}

func accessLocalAlloc() time.Duration {
	timeCh := make(chan time.Duration, CoreNum/2)
	useSumCh := make(chan int, CoreNum/2)
	set := genSet()
	for i := range CoreNum / 2 {
		go func(i int) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			if inSet(i, set) {
				println(i, "local core begin:", getCurrentCPU())
			}
			localData := make([]int, fixLen) // 512M
			start := time.Now()
			sum := 0
			for j := range N {
				localData[j%fixLen] = j
				sum += localData[j%fixLen]
			}
			elapsed := time.Since(start)
			timeCh <- elapsed
			useSumCh <- sum
			if inSet(i, set) {
				println(i, "local core finish:", getCurrentCPU())
			}
		}(i)
	}
	durations := make([]time.Duration, CoreNum/2)
	for i := range CoreNum / 2 {
		durations[i] = <-timeCh
	}
	for range CoreNum / 2 {
		<-useSumCh
	}
	return averageTime(durations)
}
