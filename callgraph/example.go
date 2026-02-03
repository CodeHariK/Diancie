package main

import (
	"errors"
	"fmt"
)

// ABC Comment
const ABC = `ABC`

// Hello Comment
type Hello struct {
	Greet int // HelloGreet
}

//Hi
/*
* How are you
* Hi you
**/
func (hello *Hello) Hi() error {
	// --->> Hi error
	return errors.New("Hello error") // Error ....
}

// PrintHello comment
func PrintHello() {
	fmt.Println("Hello")
}

/*
* Greet calls PrintHello
**/
func Greet() {
	PrintHello()
	// Greetings comment
	fmt.Println("Greetings!")
}
