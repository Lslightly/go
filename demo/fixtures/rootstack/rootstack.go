package rootstack

func outerLoopReference() {
	var outerRef *int

	for i := 0; i < 10; i++ {
		i_testOuter := i
		outerRef = &i_testOuter // 循环内部的变量 num 被外部引用，num 会逃逸到堆上
	}

	_ = outerRef
}
