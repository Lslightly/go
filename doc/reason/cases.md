# fzf

## ERReturn

/data/github_go/escapeReasonRepos/fzf/src/options.go:837:9

```go
	return func(delimiter Delimiter) func([]Token, int32) string {
		return func(tokens []Token, index int32) string {
			str := ""
			for _, holder := range parts {
				if holder.nth != nil {
					str += StripLastDelimiter(JoinTokens(Transform(tokens, holder.nth)), delimiter)
				} else if holder.index {
					if index >= 0 {
						str += strconv.Itoa(int(index))
					}
				} else {
					str += holder.str
				}
			}
			return str
		}
	}, nil
```

将function literal返回的情况


/data/github_go/escapeReasonRepos/fzf/src/options.go:3294:20


```go
	if opts.Theme.Nth.IsColorDefined() {
		return errors.New("only ANSI attributes are allowed for 'nth' (regular, bold, underline, reverse, dim, italic, strikethrough)")
	}
```

/data/github_go/escapeReasonRepos/fzf/src/algo/algo.go:655:17

```go
pos := posArray(withPos, len(pattern))
```

将如下代码内联，但是`pos`继续被return到调用者处

```go
func posArray(withPos bool, len int) *[]int {
	if withPos {
		pos := make([]int, 0, len)
		return &pos
	}
	return nil
}
```

/data/github_go/escapeReasonRepos/fzf/src/algo/algo.go:616:17 和上面的模式一模一样

/data/github_go/escapeReasonRepos/fzf/src/util/util.go:147:2

```go
func Once(nextResponse bool) func() bool {
	state := nextResponse
	return func() bool {
		prevState := state
		state = !nextResponse
		return prevState
	}
}
```

`state`变量在函数闭包中被返回

/data/github_go/escapeReasonRepos/fzf/src/options.go:3160:20

```go
	if opts.HscrollOff < 0 {
		return errors.New("hscroll offset must be a non-negative integer")
	}
```

`errors.New`

/data/github_go/escapeReasonRepos/fzf/src/options.go:935:33

```go
func parseKeyChords(str string, message string) (map[tui.Event]string, []tui.Event, error) {
	if len(str) == 0 {
		return nil, nil, errors.New(message)
	}
```

`message`通过`errors.New`传递给`error`返回值


/data/github_go/escapeReasonRepos/fzf/src/terminal.go:1417:13

```go
printFn := func(window tui.Window, limit int) {
    ...
return printFn
```

将函数闭包返回

/data/github_go/escapeReasonRepos/fzf/src/util/chars.go:54:15

```go
runes := make([]rune, bytesUntil, len(bytes))
return RunesToChars(runes)
```

转换为`Chars`后返回

/data/github_go/escapeReasonRepos/fzf/src/terminal.go:3737:41

`{storage for tl + strings.Repeat(h, repeat) + tr @ terminal.go:3737:41}`被返回


## ERGo

/data/github_go/escapeReasonRepos/fzf/src/server.go:76:75

`getHandler`和`server`绑定，`server`被传递给协程，导致逃逸。

```go
	server := httpServer{
		apiKey:        []byte(apiKey),
		actionChannel: actionChannel,
		getHandler:    getHandler,
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				continue
			}
			conn.Write([]byte(server.handleHttpRequest(conn)))
			conn.Close()
		}
	}()
```

/data/github_go/escapeReasonRepos/fzf/src/reader.go:50:7

```go
func (r *Reader) startEventPoller() {
	go func() {
		ptr := &r.event
        ...
    }
}
```

方法接收者 `r *Reader` 在方法中创建协程，并被访问


/data/github_go/escapeReasonRepos/fzf/src/terminal.go:4964:6

`{storage for func literal @ terminal.go:4964:6}` `go func`逃逸

/data/github_go/escapeReasonRepos/fzf/src/core.go:76:6

```go
prevLineAnsiState = lineAnsiState
chunkList = NewChunkList(func() {
    ansiStateDup := *prevLineAnsiState
    ...
})
reader = NewReader(func() {
    return chunkList.Push(data)
})
go reader.ReadSource()
```

通过非常复杂的数据流到并发导致的逃逸

`lineAnsiState`先赋值给`prevLineAnsiState`，`prevLineAnsiState`在某个函数闭包中使用，这个函数闭包会`chunkList`获取。`chunkList`会作为`reader`的一部分，`reader`会通过 `go reader.ReadSource` 产生并发。

/data/github_go/escapeReasonRepos/fzf/src/server.go:76:45

作为 `server` 结构体的一部分，启动另外一个协程

/data/github_go/escapeReasonRepos/fzf/src/core.go:186:21

```go
reader = NewReader(func() {
    return chunkList.Push(data)
})
go reader.ReadSource()
```

内联之后的返回值`"{storage for &Reader{...} @ core.go:186:21}"`

/data/github_go/escapeReasonRepos/fzf/src/core.go:103:2

`header`被传给 `NewChunkList`的函数闭包，也被传递给`eventBox`，用于创建`reader`

```go
header := make([]string, 0, opts.HeaderLines)
...
eventBox.Set(EvtHeader, header)
...
reader = NewReader(...,, eventBox)
go reader.ReadSource
```

/data/github_go/escapeReasonRepos/fzf/src/proxy.go:73:6

`go func() {}()`

/data/github_go/escapeReasonRepos/fzf/src/terminal.go:2704:7

```go
func (t *Terminal) printInfo() {
	if t.inputless {
		return
	}
	t.withWindow(t.inputWindow, func() { t.printInfoImpl() })
}
```

由于`t.printInfoImpl()`导致`t`由于协程导致逃逸。

`t.printInfoImpl`内部包含对`executeCommand`函数的调用，`executeCommand`函数内部有创建协程，并访问`t`

/data/github_go/escapeReasonRepos/fzf/src/core.go:69:30

`eventBox` 用于创建 `reader`，通过 `go reader.ReadSource`创建并发协程

/data/github_go/escapeReasonRepos/fzf/src/terminal.go:1060:36

```go
t := terminal{
    slab:   util.MakeSlab(slab16Size, slab32Size),
}
startHttpServer(t.dumpStatus)
```

`util.MakeSlab`内联返回值作为 `t` 的一部分。在 `startHttpServer` 中将 `dumpStatus` 方法传递给协程，导致 `t` 逃逸。导致 `slab` 并发逃逸。

/data/github_go/escapeReasonRepos/fzf/src/terminal.go:896:32

`previewBox` 作为 `t` 的一部分，由于并发导致逃逸。

/data/github_go/escapeReasonRepos/fzf/src/matcher.go:182:3

`go func (){}()` closure

/data/github_go/escapeReasonRepos/fzf/src/terminal.go:2908:7

由于调用了`t.printInfoImpl`导致`t`逃逸到堆上

/data/github_go/escapeReasonRepos/fzf/src/core.go:215:22

`patternCache`传递给`patternBuilder`函数闭包，传递给`matcher`变量，然后调用`go matcher.Loop()`函数
