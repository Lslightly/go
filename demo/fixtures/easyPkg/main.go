package main

import (
	"demo/fixtures/easyPkg/utils"
)

func main() {
	x := 1
	_ = utils.Util(&x)
	y := 1
	utils.ToHeap(&y)
}
