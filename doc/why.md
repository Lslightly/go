# whys

## assignHeap

- key of map put

`X[index]`, heap <- index

- conversion to unsafe.Pointer

`unsafe.Pointer(X)`, heap <- X

- map literal key, map literal value

```go
map[K]V{
    k: v
}
```

heap <- k, heap <- v

- send

`X <- Y`, `heap <- Y`

## assignList

- assign

`X = Y`, `X += Y`

- assign-pair

`a, b, c = x, y, z`

- assign-pair-dot-type
- assign-pair-mapr
- assign-pair-receive
- return

## deref

- appendee slice

`type(args[0]) == []*T`
`res = append(args[0], _)`, heap <-1- res

- appended slice...

`type(args[1]) == []*T`
`append(args[0], args[1]...)`

- copied slice

`type(Y) == []*T`
`copy(X, Y)`, heap <-1- Y

- indirection

`X = *Y`

- dot of pointer

`X = Y.Z`, X <-1- Y

`type(Y) == []T`
`X = Y[Z]`, X <-1-Y

- slice-to-array

`Type(X) (Type is [N]T, X is a []T)`

- reflect.Header.Data

`Y = p.Data`, k <-1- p

- range-deref

`type(Y) == []T`
`for Key, Value = range X { Body }`, Value <-1- X

## addr

- too large for stack
- too aligned for stack
- non-constant size
- reference

closure capture by ref variables X, clo <- -1 - X

- address-of

`X = &Y`, X <- -1 - Y

- spill

`X = new(int)`, X <- -1 - new(int)

## dotType

- dot

`X.(T)`

`type(T) is interface`, k <- X
`type(T) is not interface`, k <-1- X

- switch case

type switch

`case T`


## other


call parameter
callee operand
//go:uintptrescapes
captured by a closure
dot
fixed-array-index-of
slice
interface-converted
array literal element
slice-literal-element
struct literal element
range

