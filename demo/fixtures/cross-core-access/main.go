package main

import (
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

const (
	N       int = 3_000_000_000 // 访问次数
	fixLen  int = 1024
	CoreNum int = 256
)

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
	timeCh := make(chan time.Duration, CoreNum-1)
	useSumCh := make(chan int, CoreNum-1)
	for range CoreNum - 1 {
		remoteData := make([]int, fixLen) // 4K
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			start := time.Now()
			sum := 0
			for j := range N {
				remoteData[j%fixLen] = j
				sum += remoteData[j%fixLen]
			}
			elapsed := time.Since(start)
			timeCh <- elapsed
			useSumCh <- sum
		}()
	}
	durations := make([]time.Duration, CoreNum-1)
	for i := range CoreNum - 1 {
		durations[i] = <-timeCh
	}
	for range CoreNum - 1 {
		<-useSumCh
	}
	return averageTime(durations)
}

func accessLocalAlloc() time.Duration {
	timeCh := make(chan time.Duration, CoreNum-1)
	useSumCh := make(chan int, CoreNum-1)
	for range CoreNum - 1 {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			localData := make([]int, fixLen) // 4K
			start := time.Now()
			sum := 0
			for j := range N {
				localData[j%fixLen] = j
				sum += localData[j%fixLen]
			}
			elapsed := time.Since(start)
			timeCh <- elapsed
			useSumCh <- sum
		}()
	}
	durations := make([]time.Duration, CoreNum-1)
	for i := range CoreNum - 1 {
		durations[i] = <-timeCh
	}
	for range CoreNum - 1 {
		<-useSumCh
	}
	return averageTime(durations)
}
