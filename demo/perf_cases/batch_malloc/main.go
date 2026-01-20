//go:build linux && batch

package main

import (
	"reflect"
	"runtime"
	"unsafe"
)

type Obj struct {
	a, b int
}

// emptyInterface 是 interface{} 的运行时表示
type emptyInterface struct {
	typ  unsafe.Pointer
	word unsafe.Pointer
}

func GetRuntimeTypePtr(t reflect.Type) unsafe.Pointer {
	// reflect.Type 本身是一个接口
	// 在 reflect 包中，所有的类型（如 rtype）都实现了这个接口
	// 我们通过 unsafe.Pointer 强制转换来读取接口内部指向真实类型信息的指针
	eface := *(*emptyInterface)(unsafe.Pointer(&t))

	// 对于 reflect.Type 接口，eface.word 指向的是具体的类型描述符（rtype）
	return eface.word
}

func foo() {
	s := make([]*Obj, 10)
	runtime.BatchEnter()
	for range 10 {
		obj := (*Obj)(runtime.BatchNewobject(GetRuntimeTypePtr(reflect.TypeFor[Obj]())))
		s = append(s, obj)
	}
	runtime.BatchExit()
}

func main() {
	for i := range 1000 {
		foo()
		println(i)
	}
}
