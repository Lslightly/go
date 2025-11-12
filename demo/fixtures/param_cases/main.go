package paramcases

var sink interface{}
var sink2 interface{}

//go:noinline
func t1(p *int) { // p ERParamLeak|ERGlobal, no ERHeap
	sink = p
}

//go:noinline
func t2(p *int) {
	sink = *p // {storage for *p @ main.go:11:9} ERGlobal|ERHeap
}

//go:noinline
func t3(p *int) { // p ERGlobal|ERHeap, no ERParamLeak
	sink = &p
}

//go:noinline
func t4(p *int) *int { // p ERParamLeak|ERGlobal|ERReturn, no ERHeap
	sink = p
	return p
}

//go:noinline
func t5(p *int) { // p ERParamLeak|ERGlobal|ERHeap
	sink = p
	sink2 = &p
}

//go:noinline
func t6(p **int) { // p ERParamLeak|ERGlobal, no ERHeap
	sink = *p
}

func caller1() {
	p := new(int) // new(int) ERGlobal|ERHeap
	t1(p)
}

func caller2() {
	p := new(int) // new(int) ERGlobal|ERHeap, no ERParamLeak
	t3(p)
}

func caller3() {
	pi := new(int) // new(int) ERGlobal|ERHeap, no ERParamLeak
	t6(&pi)
}
