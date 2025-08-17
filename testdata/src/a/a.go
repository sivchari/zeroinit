package a

func f() {
	// Test cases for zero value initialization
	s := ""               // want "should use var declaration for zero value of string"
	i := 0                // want "should use var declaration for zero value of int"
	f := 0.0              // want "should use var declaration for zero value of float64"
	b := false            // want "should use var declaration for zero value of bool"
	slice := []int{}      // want `should use var declaration for zero value of \[\]int`
	m := map[string]int{} // want `should use var declaration for zero value of map\[string\]int`

	// These should NOT trigger warnings
	s2 := "hello"
	i2 := 42
	f2 := 3.14
	b2 := true
	slice2 := []int{1, 2, 3}
	m2 := map[string]int{"key": 1}

	// Use variables to avoid unused variable errors
	_, _, _, _, _, _ = s, i, f, b, slice, m
	_, _, _, _, _, _ = s2, i2, f2, b2, slice2, m2
}

func g() {
	// More test cases
	slice3 := make([]byte, 0) // want `should use var declaration for zero value of \[\]byte`

	// These should NOT trigger warnings (channels need make)
	ch := make(chan int)
	ch2 := make(chan int, 0)
	ch3 := make(chan int, 1)
	slice4 := make([]byte, 10)

	// Use variables
	_, _, _, _ = ch, ch2, ch3, slice3
	_ = slice4
}

func h() {
	// Pointer cases
	// p := nil would not trigger warning because type cannot be determined
	// These are the correct ways to declare nil pointers:
	var p1 *int       // Good: using var for nil pointer
	p2 := (*int)(nil) // OK: explicit type cast
	p3 := &struct{}{} // OK: not a zero value, it's an address

	// Interface cases
	var err error // Good: using var for nil interface

	// Use variables
	_, _, _ = p1, p2, p3
	_ = err
}
