package src_test

import (
	"fmt"

	"github.com/mmcloughlin/avo/src"
)

func ExamplePosition_IsValid() {
	fmt.Println(src.Position{"a.go", 42}.IsValid())
	fmt.Println(src.Position{"", 42}.IsValid())
	fmt.Println(src.Position{"a.go", -1}.IsValid())
	// Output:
	// true
	// true
	// false
}

func ExamplePosition_String() {
	fmt.Println(src.Position{"a.go", 42})
	fmt.Println(src.Position{"", 42})
	fmt.Println(src.Position{"a.go", -1}) // invalid
	// Output:
	// a.go:42
	// 42
	// -
}
