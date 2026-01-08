[blog post](https://nghiant3223.github.io/2025/04/15/go-scheduler.html)

> 这篇文章涉及GC不多。

协作式抢占和异步抢占均通过 sysmon 线程触发，协作式抢占由 sysmon 线程设置 `stackguard0` 为 `stackPreempt`，异步抢占由 sysmon 向 M 的singal handler发送 SIGURG 信号。

调度器 `findRunnable` 是查找可执行协程的关键，具体寻找顺序见 [Finding a Runnable Goroutine](https://nghiant3223.github.io/2025/04/15/go-scheduler.html#finding-a-runnable-goroutine)。

GC 4个阶段：
1. First STW: 所有的P都到底safe point
2. Marking Phase: GC协程 获得 P，来标记可达对象。
3. Second STW: 进程暂停，来结束标记阶段
4. Sweeping Phase: 恢复进程执行，在后台回收不可达对象。

---

> > In order to avoid starving goroutines in a blocked M’s local run queue like M1 in the figure above, the scheduler should allow other threads to steal goroutine from it ^bk336ma

> > However, with a large number of blocked threads, scanning all of them to find a non-empty run queue becomes expensive. ^bk336m3

> > M is detached from P when G makes system call, the memory consumption stays low when there are a large number of Ms entering system calls. ^bk336mb

> > When a goroutine finishes execution, it isn’t destroyed; instead, it becomes dead and is placed into the free list of the current processor P .
> > If P’s free list is full, the dead goroutine is moved to the global free list.
> > When a new goroutine is created, the scheduler first attempts to reuse one from the free list before allocating a new one from scratch.
> > This recycling mechanism makes goroutine creation significantly cheaper than creating a new thread.
> goroutine数据结构的复用使得协程的创建代价相比于new thread的创建要低很多 ^bk336m8

> > In addition to stack growth and shrinkage, operations like garbage collection and parking a goroutine also require execution on the system stack.
> > Whenever a thread performs such operation, it switches to the system stack and executes the operation in the context of g0.
> 需要在系统栈(g0运行的栈)上执行的操作，包括:
> 1. stack growth/shrinkage
> 2. garbage collection
> 3. parking a goroutine ^bk336mh

> > threads run scheduler code as soon as M is created, therefore the initial state of M is running. ^bk336m2

> > When M is created or woken up, the scheduler guarantees that there is always an idle processor P so that it can be associated with M to run Go code ^bk336mf

> > Note that not all M enters spinning state, it does so only if the number of spinning threads is less than half of the number of busy processors ^bk336m9

> > A local run queue is not just a queue but composed of two components: runnext which holds a single prioritized goroutine, and runq which is a queue of goroutines ^bk336m6

> > The runq component of P is an array-based, fixed-size, and circular queue. ^bk336mc

> > pageCache, on the other hand, enables the memory allocator to fetch memory pages without acquiring the heap lock, thereby improving performance under high concurrency. ^bk336m5

> > When looking for a runnable goroutine, P also checks if there are any timers that have expired.
> > If so, P adds the corresponding goroutine with timer to its local run queue, giving chance for the goroutine to run. ^bk33683

> > It’s worth noting that the main thread, M0, is responsible not only for running the main goroutine but also for executing other goroutines ^bk3368h

> > If runnext already contains a goroutine G1, the scheduler attempts to move G1 to runq—provided runq is not full—and put G into runnext
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L6771-L6783
> 当runqputshow返回false时，代表pp.runqhead被移动，有空余的空间给G1，这时候G可以被放到runnext上，G1可以移动到runq中 ^bk33687

> > If runq is full, G1 along with half of the goroutines in runq are moved to the global run queue to reduce the workload for P.
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L6788-L6825 ^bk3368d

> > When a new goroutine is created, and we aim to maximize program concurrency, the thread which goroutine is running on attempts to wake up another processor P by futex system call.
> 这里的`futex` system call是通过`wakep`调用实现的。`wakep`会调用`startm`函数，`startm`会通过`notewakeup`函数唤醒OS线程。`notewakeup`函数就会通过`futexwakeup`函数唤醒OS线程。
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3020
> https://github.com/golang/go/blob/master/src/runtime/lock_futex.go#L32
> 每次通过`newproc`创建一个新协程（在main协程起来之后），都会调用`wakep`函数，此时有可能会找到正处于睡眠状态的M，也有可能会新创建一个M和p绑定
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L5026-L5028 ^bk3368e

> > ![](https://g1proxy.wimg.site/seN2lY5qDCzamRC09kdgXx-GokmfymbNIQzwYWwQeGGE=/https://nghiant3223.github.io/assets/2025-03-11-go-scheduling/creating_goroutine.png) 
> 当协程加到259个时，M0正在执行G1协程，next原本是G258, runq中原本是G2-G257共256个协程。此时会将runq中一般的协程放到globrunq中，即G131-G257共127个协程加上G258放到globrunq中。并将G259放到runnext中 ^bk33e8m

> > The schedule function in the Go runtime is responsible for finding and executing a runnable goroutine. ^bk33e8d

> > ![](https://g1proxy.wimg.site/svm5xxHC3320-R3hnYMzYANInWHVqhPXh7sNypWBDir0=/https://nghiant3223.github.io/assets/2025-03-11-go-scheduling/processor_state_machine.png) 
> P state machine ^bk33e7h

> > ![](https://g1proxy.wimg.site/sVgji3Gzmg0k9uqKQWDKsrsIfxe3NShOGhBrt3UBhgkU=/https://nghiant3223.github.io/assets/2025-03-11-go-scheduling/thread_state_machine.png) 
> M state machine ^bk33e72

> > ![](https://g1proxy.wimg.site/skM8JONZ-xY99DXkQgvoQBibKH6BVvI5Bfde464XZ_Kc=/https://nghiant3223.github.io/assets/2025-03-11-go-scheduling/goroutine_state_machine.png) 
> G state machine ^bk33e7f

> > various scenarios
> 除了以下几个之外，还包括goyield ^bk33edm

> > when a new thread is created
> mstart1 https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L1862 ^bk33e75

> > when Gosched is called
> gosched_m https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4179 ^bk33eda

> > when a goroutine is parked or preempted
> park_m https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4141
> 抢占preemptPark
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4251 ^bk33edk

> > after a goroutine completes a system call
> exitsyscall0 https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4875 ^bk33ed2

> > returns
> goexit0 https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4310 ^bk33edh

> > Once a goroutine is selected, it transitions from runnable to running state, signaling that it’s ready to run
> 通过`casgstatus(gp, _Grunnable, _Grunning)`将协程G的状态修改为`_Grunning` ^bk33ef3

> > a kernel thread invokes the gogo function to begin goroutine execution.
> 通过`gogo(&gp.sched)`开始协程的执行 ^bk33eff

> > As described in the Initializing Goroutine section, when a goroutine completes, the goexit function is invoked.
> > This function eventually leads to a call to goexit0, which performs cleanup for the terminating goroutine and re-enters the schedule function—bringing the schedule loop back.
> 当一个协程结束执行时，会返回到`goexit`栈，然后调用`goexit0`函数，最终调用`schedule`函数，形成一个大循环。`schedule`函数会利用`gp.sched`结构体将寄存器切换协程对应的sp, pc等，然后继续执行。每个协程在创建时就自动分配了2K大小的栈。切换路径为`execute`->`gogo(&gp.sched)`.
> https://github.com/golang/go/blob/go1.24.0/src/runtime/asm_amd64.s#L404
> 在`gogo`中，会通过`MOVQ DX, g(CX)`和`MOVQ DX, R14`将`gp`指针存到TLS中，相当于告诉M当前执行的协程 ^bk33e52

> > Just take a look at the main function in Go runtime
> `runtime.main`函数由main goroutine执行，不一定由main thread执行 ^bk33e68

> > That’s how the process can exit and the reason why the main goroutine doesn’t wait for goroutines spawned by go keyword.
> 当主协程执行完`main_main`函数之后，会调用`exit(0)`系统调用，退出进程，不会等待其他协程执行结束再退出。 ^bk33e6b

> > 1/61 of the time,
> 通过`pp.schedtick%61 == 0`实现。
> 如果没有这个检查，那么两个协程通过相互spawn，可以完全占满local runqueue，没有公平性 ^bk335da

> > Check local run queue of the associated processor P if M is spinning.
> 前面还有一步是唤醒finalizer G
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3339-L3342 ^bk335d7

> > Steal from other processors P1’s local run queue.
> 通过`stealWork`实现。https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3676。感觉有点难以理解 ^bk33558

> > Check garbage collection worker goroutine’s availability again.
> `gp = gcBgMarkWorkerNode.gp.ptr()` ^bk335ca

> > Step 3, 5 and 9 don’t just take one goroutine but attempts to grab a batch for better efficiency.
> globrunqget
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L6595-L6598。通过在`globrunqget`中循环`runqput`实现批量将全局队列中的元素放到local runq中，并且获取一个gp返回。 ^bk335cf

> > (global_queue_size/number_of_processors)+1
> `sched.runqsize/gomaxprocs+1` ^bk335cd

> > won’t take more than half of the P’s local queue capacity
> len(pp.runq)/2 ^bk335ce

> > Step 4
> `runqget` https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L6861-L6886 ^bk335cc

> > Step 7
> `stealWork` https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3376-L3402 ^bk3356a

> > During the first three attempts, it tries to steal goroutines only from P1’s runq
> `runqsteal` https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3730-L3736
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L6983-L7003
> 前3次尝试从P2 local runq中偷 ^bk3356m

> > half of the goroutines from P1’s runq are transferred to the current processor P’s runq
> 循环赋值，从p2的`runq`中移动n份G ^bk33566

> > On the last attempt
> last attempt中`steadRunNextG`为true。https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L6938 ^bk335eb

> > Note that findRunnable not only finds a runnable goroutine but also wakes up goroutine that went into sleep before step 1 happens.
> > Once the goroutine wakes up, it’ll be put into the local run queue of the processor P that was executing it, waiting to be picked up and executed by some thread M.
> 这里并非通过`notewakeup`唤醒协程，而是通过`netpoll`之后获得glist，然后通过`injectglist`将glist放到相应的队列中。先在globrunq中放置npidle个数的协程，然后在runq中放置剩下的协程。然后通过`wakep`将P和M绑定 ^bk335e6

> > thread M waits on netpoll until the nearest timer expires
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3579-L3580 ^bk33cbb

> > Go’s timer system heavily relies on netpoll, as noted in this code comment ^bk33cbd

> > After netpoll returns
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3602 ^bk33cb6

> > the stopm function ^bk33cc3

> > Non-cooperative preemption: a too long-running goroutine is forced to stop.
> > Cooperative preemption: a goroutine voluntarily yields its processor P.
> 两类抢占机制 ^bk33ck7

> > In Go runtime, there is a daemon running on a dedicated thread M without a P, called sysmon (i.e. system monitor).
> sysmon守护进程运行在特定的M上，不需要P ^bk33cmk

> > it signals thread M by executing tgkill system call to forcefully preempt the running goroutine.
> 通过preemptM执行tgkill通知线程。这里的发送信号方式是通过`getpid`获取当前thread group的进程id tgid，然后发送sig给thread id(tid) mp.procid。mp.procid就是M所属OS线程id ^bk33cmm

> > Upon receiving SIGURG, the execution of the program is transferred to the signal handler,
> 接收到`SIGURG`信号之后，程序执行就被转移到signal handler上了。这个handler通过`initsig`函数注册。 ^bk33cm9

> > The execution switch from main program to signal handler is triggered by the kernel4,5. ^bk33c8k

> > In asyncPreempt2, the goroutine g0 of thread M will enter gopreempt_m to disassociate goroutine G from M and enqueue G into the global run queue.
> 通过`goschedImpl`将协程G放到全局队列中。
> 在通过preempt之后，preempt之前的状态还保留在栈中，通过`asyncPreempt2`就调用了`gopreempt_m`函数，然后执行`schedule`函数执行协程。这时候会使用新协程的栈帧。
> `gopreempt_m`会将协程G状态从_Grunning切换为_Grunnable，然后通过`dropg`将G和M解绑，然后将协程放在globrunq中。
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4161-L4166 ^bk33c8h

> > As preemption signal is triggered by sysmon but the actual preemption doesn’t happen until the thread receives preemption signal, this kind of preemption is asynchronous.
> > That’s why goroutines can actually run beyond the time limit 10ms, like goroutine G9 in the example.
> 异步抢占导致协程可能会跑超过时间限制的时间 ^bk33c8c

> > ![](https://g1proxy.wimg.site/s0SZpRMmdJ6T4JodxcGG-oGCOtLrCpAeIqYHTLkRWhnE=/https://nghiant3223.github.io/assets/2025-03-11-go-scheduling/non_cooperative_preemption.png) 
> G1是如何sleep的？通过timeSleep函数(https://github.com/golang/go/blob/go1.24.0/src/runtime/time.go#L338)调用gopark，将协程从_Grunning到_Gwaiting状态，并且通过park_m将G1和M解绑，通过schedule找到新的协程执行。
> 在协程被异步抢占之后，schedule调用findRunnable实现对G1的唤醒。具体唤醒方式为`checkTimersNoP`检查timer是否执行结束。
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3576
> 在获得pollUntil之后，通过`netpoll(delay)`等待一定的时间，然后会返回一个`list`和`delta`，基于`list`找到可运行的协程，并将其他协程`injectglist`
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3602-L3632
> 但是这个协程是怎么加到runq中的？在go1.24中讲timer和netpoll做了整合。在图中的情况下，由于没有空闲的P，所以会直接通过`injectglist`将协程全部放到runq中。
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3617-L3618 ^bk33c78

> > We, as Go programmers, had to tell goroutines to cooperatively give up its processor P by making a call to runtime.Gosched() in the loop body.
> > There was a Stackoverflow question that described an example and the behavior of runtime.Gosched().
> 最早的时候需要通过`runtime.Gosched`函数调用手动执行合作式抢占 ^bk33ce5

> > R16
> stackguard0
> https://github.com/golang/go/blob/go1.24.0/src/runtime/runtime2.go#L399-L405 ^bk339bc

> > R17
> RSP-48 ^bk3393h

> > The stack guard is a special value placed at the end of the stack. When the stack pointer reaches this value, Go runtime detects that the stack is nearly full and needs to grow—that’s exactly what the next three instructions do. ^bk33933

> > runtime.morestack_noctxt function will eventually call newstack to grow the stack and optionally enter gopreempt_m to trigger preemption as discussed in non-cooperative preemption.
> > The key point of cooperative preemption is the condition for entering gopreempt_m, which is stackguard0 == stackPreempt.
> 当`stackguard0 == stackPreempt`时，就会实现协作式抢占。由于stackPreempt是一个很大的值，所以在函数的序言prologue部分一定会进入morestack_noctxt调用，然后就会实现合作式抢占。但其实也是sysmon通知协程跑了一段时间该退出了，协程刚好调用函数了，就被调度了。 ^bk3393c

> > stackPreempt can be set by the sysmon if a goroutine has been running for more than 10ms
> `stackPreempt`标记由sysmon设置，协程在调用函数时如果发现有，就主动让出P ^bk3mh33

> > The goroutine will then be cooperatively preempted if it makes a function call or non-cooperatively preempted by the thread’s signal handler, whichever happens first.
> 有两种抢占方式，一种是signal异步抢占，一种是协程调用新函数，然后进入newstack，主动让出 ^bk339ka

> > ![](https://g1proxy.wimg.site/sqd7sCsvwu4SmjlPYdwYCFWZtjjMvqOUU6TmfUG-NXo4=/https://nghiant3223.github.io/assets/2025-03-11-go-scheduling/runtime_trace_cooperative_preempt.png) 
> 在由sysmon设置stackguard0为stackPreempt之后，协程主动让出P ^bk339m3

> > Go’s runtime employs an M:N threading model, further optimized by the use of logical processors P, making its approach to handling system calls particularly interesting.
> M:N线程模型，通过逻辑处理器进一步优化，使得系统调用变得很有趣。 ^bk3madc

> > RawSyscall and Syscall
> RawSyscall会直接调用系统调用，而Syscall则会先通过runtime_entersyscall()和runtime_exitsyscall()更改运行时状态。
> https://github.com/golang/go/blob/go1.24.0/src/syscall/syscall_linux.go#L72-L89
> runtime.entersyscall
> https://github.com/golang/go/blob/d1d0fc7a97539206e33f04bac935f6450597137c/src/runtime/proc.go#L4737 ^bk3ma9k

> > Syscall is typically used for operations with unpredictable durations, such as reading from a file or writing an HTTP response
> syscall主要调用性能不可预测的函数 ^bk3ma9a

> > f a P remains in syscall state for more than 10ms, instead of non-cooperatively preempting the running goroutine, a processor handoff takes place.
> 这段代码不知道在哪里实现？ ^bk3ma99

> > while system call is still in progress and whether sysmon happens to seize P or not, the association between goroutine G and thread M still remains.
> 当P还在系统调用时，G和M之间的关联还是存在的。 ^bk3ma9c

> > A thread M makes system call on behalf of some goroutine G, that’s why the association between them is maintained as-is. ^bk3mafk

> > Another important point is that whenever a processor P is in syscall state, it can't be taken up by another thread M to execute code until sysmon happens to seize it or until the system call is completed
> 当一个P在syscall状态时，不能绑定到其他M上，除非sysmon抓到了它或者系统调用结束。
> 这里的“抓到了它”是什么意思？大概意思是sysmon发现P在Psyscall状态很长时间（超过10ms），通过retake函数将P从Psyscall状态切换为Pidle状态了。 ^bk3mafa

> > Therefore, in case there are multiple system calls happening at the same time, the program (excluding system calls) doesn’t make any progress.
> 如果有很多系统调用，那么除了系统调用之外的程序就无法执行，因为P总共最多就是GOMAXPROCS这么多。 ^bk3maf2

> > if is a processor P available to execute the goroutine G that has just completed its system call.
> > This P can either be the same one that previously executed G, if it is still in the syscall state (i.e., it hasn’t been seized by sysmon), or any other processor P1 currently in the idle state—whichever is found first.
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4757
> 以上代码是处于syscall状态的P
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4771-L4779
> 以上代码是处于idle状态的P ^bk3mb7h

> > Before the fast path exits, G transition from syscall state to running state.
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4706 ^bk3mb7f

> > retrieving any idle processor P once again
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4843-L4846 ^bk3mb92

> > If it’s found, the goroutine G is scheduled to run on that P
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4862-L4865 ^bk3mb9f

> > G is enqueued into the global run queue
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4848-L4856 ^bk3mb97

> > the associated thread M is stopped by stopm function, waiting to be woken up to continue the schedule loop.
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4874-L4875 ^bk3mb9c

> > notorious
> 臭名昭著的 ^bk3mb5k

> > C10K.
> C10K问题：优化网络栈以在同一时间服务更多客户。 ^bk3mbfe

> > Similarly, http.HandleFunc() registers your handler functions, abstracting away the lower-level details like using read() system call to read data, and write() system call to write data to the network socket.
> 通过http.ResponseWriter抽象write, 通过http.Request抽象read ^bk3mb52

> > Blocking I/O is simpler to implement but inefficient, as it requires the application to spawn N threads for N connections. In contrast, non-blocking I/O is more complex, but when implemented correctly, it enables significantly better resource utilization.
> 阻塞IO和非阻塞IO的区别。非阻塞IO可以只有一个connection处理多个IO ^bk3mb59

> > Another I/O model worth mentioning is I/O multiplexing, in which select, or poll system call is used to wait for one of a set of file descriptors to become ready to perform I/O
> 等待一组file descriptor中的一个ready ^bk3mb5d

> > When a TCP listener accepts a connection, accept4 system call is invoked with SOCK_NONBLOCK flag to set the socket’s file descriptor of the socket to non-blocking mode
> 通过`SOCK_NONBLOCK`将file descriptor设置为non-blocking mode. ^bk3mb66

> > The epoll instance is initialized in the poll_runtime_pollServerInit function, which is wrapped in a sync.Once to ensure it runs only once.
> https://github.com/golang/go/blob/go1.24.0/src/internal/poll/fd_poll_runtime.go#L39
> sync.Once.Do，而且sync.Once是全局变量，保证整个进程只会创建一个epoll instance ^bk3mbef

> > poll_runtime_pollServerInit function,
> 该函数用于初始化 `netpollInitLock`。并将 `netpollInited` 设置为1. ^bk3mbe3

> > As epoll monitors file descriptors rather than goroutines, epoll_ctl also associates the file descriptor with an instance of runtime.pollDesc, allowing the Go scheduler to identify which goroutine should be resumed when I/O readiness is reported.
> `epoll_ctl`还将file descriptor和`runtime.pollDesc`关联起来，允许Go调度器识别哪些协程在IO ready的时候需要被恢复 ^bk3m33a

> > If the I/O data is not ready yet, i.e. EAGAIN error is returned,
> https://github.com/golang/go/blob/go1.24.0/src/internal/poll/fd_unix.go#L164-L168 ^bk3m3df

> > Go runtime invokes poll_runtime_pollWait method to park the goroutine.
> gopark的位置可能是netpollblock中
> https://github.com/golang/go/blob/go1.24.0/src/runtime/netpoll.go#L574-L576 ^bk3m3d9

> > In netpoll function, epoll_wait system call is used to monitor up to 128 file descriptors in a specified amount of time.
> https://github.com/golang/go/blob/go1.24.0/src/runtime/netpoll_epoll.go#L119 ^bk3m3dd

> > Finally, netpoll extracts the goroutine references from runtime.pollDesc and hands them off to the Go runtime.
> 通过调用 `netpollready` 将协程添加到 `toRun` 队列中。
> https://github.com/golang/go/blob/go1.24.0/src/runtime/netpoll_epoll.go#L171
> `netpollready` 源码 https://github.com/golang/go/blob/master/src/runtime/netpoll.go#L494-L510 ^bk3m3d7

> > It’s triggered when a thread looks for a runnable goroutine to execute, as outlined in schedule loop
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L3602 ^bk3m3d6

> > As mentioned above, epoll instance monitors up to 128 file descriptors.
> > Therefore, it’s important to unregister file descriptors when they are no longer needed otherwise some goroutines may be starved.
> > When file or network connection is no longer in used, we should close it by calling its Close method.
> `epoll`最多监视128个文件描述符，因此关闭文件描述符很重要。 ^bk3m3dc

> > make epoll_ctl with EPOLL_CTL_DEL operation
> https://github.com/golang/go/blob/master/src/runtime/netpoll_epoll.go#L57-L60 ^bk3m393

> > Sweeping phase: Unpause the process and reclaim memory for unreachable objects in background.
> sweep阶段在后台并发执行 ^bk3m3ka

> > But when is the current goroutine stored in thread-local storage so it can be retrieved later?
> > This happens during a goroutine context switch in the gogo function, which is called by execute.
> `gogo`函数设置当前的goroutine
> https://github.com/golang/go/blob/go1.24.0/src/runtime/asm_amd64.s#L411-L413 ^bk3m379

> > This is a commonly used procedure in Go runtime for transitioning the current goroutine into a waiting state and scheduling another goroutine to run.
> 将当前协程的状态转移到waiting，并且调度其他协程运行 ^bk3m3dk

> > If unlockf returns false
> `unlockf` returns false.
> https://github.com/golang/go/blob/go1.24.0/src/runtime/proc.go#L4123-L4134 ^bk3m3db

> > transferring the ownership of a processor P from a thread Ms that is blocking in a system call to another thread M1 ^bk3ma9h

> > P will be associated with M1 to make progress by calling startm under certain conditions: if the global run queue is not empty, if its local run queue is not empty, if there is tracing work or garbage collection work to do, or if no thread is currently handling netpoll.
> > If none of these conditions is met, P is returned to the processor idle list
> P和M1通过调用`startm`关联有几个条件:
> 1. globrunq非空
> 2. P local runq非空
> 3. tracing work, GC work to do
> 4. no thread currently handling netpoll ^bk3ma92

> > ![](https://g1proxy.wimg.site/sbZ_9ru0n9c7HKycQ756NTBZPsTk2Di0tv6MOWmPEIx8=/https://nghiant3223.github.io/assets/2025-03-11-go-scheduling/syscall_fast_path_1.png) 
> syscall completes before sysmon seizes P0 ^bk3mb7d

> > ![](https://g1proxy.wimg.site/s75LAbhefEoLO_dYD-SfSMhn3ca4qFFEe_am0YzcZEbU=/https://nghiant3223.github.io/assets/2025-03-11-go-scheduling/syscall_fast_path_2.png) 
> sysmon seizes and retakes P0, handoff happening, P0 is bound to M2.
> Then syscall completes, M0 is bound to another P, P1. ^bk3mb7e


