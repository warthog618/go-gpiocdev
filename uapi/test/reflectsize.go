package main

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/warthog618/go-gpiocdev/uapi"
)

// sizeof returns the size >= 0 of variables for the given type or -1 if the type is not acceptable.
func sizeof(t reflect.Type) int {
	switch t.Kind() {
	case reflect.Array:
		if s := sizeof(t.Elem()); s >= 0 {
			return s * t.Len()
		}

	case reflect.Struct:
		sum := 0
		for i, n := 0, t.NumField(); i < n; i++ {
			s := sizeof(t.Field(i).Type)
			if s < 0 {
				return -1
			}
			sum += s
		}
		return sum

	case reflect.Bool,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return int(t.Size())
	}

	return -1
}

// dataSize returns the number of bytes the actual data represented by v occupies in memory.
// For compound structures, it sums the sizes of the elements. Thus, for instance, for a slice
// it returns the length of the slice times the element size and does not count the memory
// occupied by the header. If the type of v is not acceptable, dataSize returns -1.
func dataSize(v reflect.Value) int {
	if v.Kind() == reflect.Slice {
		if s := sizeof(v.Type().Elem()); s >= 0 {
			return s * v.Len()
		}
		return -1
	}
	return sizeof(v.Type())
}

func main() {
	var lic uapi.LineInfoChanged

	fmt.Printf("unsafe.sizeof: %d\n", unsafe.Sizeof(lic))

	v := reflect.ValueOf(lic)
	size := dataSize(v)
	fmt.Printf("reflect size: %d\n", size)

	fmt.Printf("unsafe.sizeof: %d\n", unsafe.Sizeof(lic.Info))
	v = reflect.ValueOf(lic.Info)
	size = dataSize(v)
	fmt.Printf("reflect size: %d\n", size)
}
