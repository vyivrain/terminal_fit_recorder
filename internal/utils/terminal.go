package utils

import "fmt"

// ClearScreen clears the terminal screen using ANSI escape codes
func ClearScreen() {
	fmt.Print("\033[2J\033[H")
}
