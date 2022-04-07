package logdump

//package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

// Println is a function used to log with timestamp
func Println(msg string) {
	color.Set(color.FgGreen)
	fmt.Printf("INFO ")
	color.Unset()
	color.Set(color.FgMagenta)
	fmt.Printf(time.Now().Format("15:04:05 "))
	color.Unset()
	fmt.Print(msg)
}

// DefaultPrint is a wrapeer around Println in fmt package
func DefaultPrintBlue(a ...interface{}) {
	color.Set(color.FgBlue)
	fmt.Print(a...)
	color.Unset()
}

func DefaultPrint(a ...interface{}) {
	fmt.Print(a...)
}

func Printtx(msg string) {
	color.Set(color.FgBlue)
	fmt.Printf(msg)
	color.Unset()
}

// func main() {
// 	DefaultPrintBlue("Hello,World !")
// }
