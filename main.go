package main

import "fmt"
import "github.com/stianeikeland/go-rpio"

func main() {
	fmt.Println("foo")

	err := rpio.Open()
	defer rpio.Close()
}
