package demo

import "demo/indirect"

func ImportPkgs() {
	indirect.Indirect()
	indirect.Dotptr()
	indirect.Mapidx()
}
