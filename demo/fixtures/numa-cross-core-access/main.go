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

	"golang.org/x/sys/unix"
)

const (
	N       int = 1_000_000_000        // 访问次数
	fixLen  int = 64 * 1024 * 1024 / 8 // 64MiB []int, L3 cache is 32MiB
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
	// set := genSet()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	setCpu(CoreNum / 2)
	for i := range CoreNum / 2 {
		remoteData := make([]int, fixLen)
		go func(i int) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			setCpu(i)
			start := time.Now()
			sum := 0
			for j := range N {
				remoteData[j%fixLen] = j
				sum += remoteData[j%fixLen]
			}
			elapsed := time.Since(start)
			timeCh <- elapsed
			useSumCh <- sum
			assertCpuAffinity(i)
		}(i)
	}
	assertCpuAffinity(CoreNum / 2)
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

func setCpu(cpu int) {
	cpuSet := unix.CPUSet{}
	cpuSet.Set(cpu)
	unix.SchedSetaffinity(0, &cpuSet)
}

func assertCpuAffinity(expected int) {
	cpuSet := unix.CPUSet{}
	err := unix.SchedGetaffinity(0, &cpuSet)
	if err != nil {
		log.Fatal(err)
	}
	if !cpuSet.IsSet(expected) {
		log.Fatalf("expected cpu %d, got %v", expected, cpuSet)
	}
}

func accessLocalAlloc() time.Duration {
	timeCh := make(chan time.Duration, CoreNum/2)
	useSumCh := make(chan int, CoreNum/2)
	// set := genSet()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	setCpu(CoreNum / 2)
	for i := range CoreNum / 2 {
		go func(i int) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			setCpu(i)
			localData := make([]int, fixLen)
			start := time.Now()
			sum := 0
			for j := range N {
				localData[j%fixLen] = j
				sum += localData[j%fixLen]
			}
			elapsed := time.Since(start)
			timeCh <- elapsed
			useSumCh <- sum
			assertCpuAffinity(i)
		}(i)
	}
	assertCpuAffinity(CoreNum / 2)
	durations := make([]time.Duration, CoreNum/2)
	for i := range CoreNum / 2 {
		durations[i] = <-timeCh
	}
	for range CoreNum / 2 {
		<-useSumCh
	}
	return averageTime(durations)
}
