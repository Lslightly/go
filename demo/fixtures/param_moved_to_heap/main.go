package parammovedtoheap

func ParamMovedToHeapGo(x int) {
	c := make(chan bool)
	go func() { x = 2; c <- true }()
	<-c
}

func defer1(i int) (x int) {
	c := make(chan int)
	go func() { x = i; c <- 1 }()
	<-c
	return
}
