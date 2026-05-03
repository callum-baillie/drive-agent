package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	// ANSI color codes
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	// Status symbols
	SymbolCheck = "✓"
	SymbolCross = "✗"
	SymbolWarn  = "⚠"
	SymbolInfo  = "ℹ"
	SymbolArrow = "→"
	SymbolDot   = "●"
)

// Success prints a green success message.
func Success(msg string, args ...interface{}) {
	fmt.Printf(Green+SymbolCheck+" "+msg+Reset+"\n", args...)
}

// Error prints a red error message.
func Error(msg string, args ...interface{}) {
	fmt.Printf(Red+SymbolCross+" "+msg+Reset+"\n", args...)
}

// Warning prints a yellow warning message.
func Warning(msg string, args ...interface{}) {
	fmt.Printf(Yellow+SymbolWarn+" "+msg+Reset+"\n", args...)
}

// Info prints a blue info message.
func Info(msg string, args ...interface{}) {
	fmt.Printf(Blue+SymbolInfo+" "+msg+Reset+"\n", args...)
}

// Header prints a bold header.
func Header(msg string, args ...interface{}) {
	fmt.Println()
	fmt.Printf(Bold+msg+Reset+"\n", args...)
	fmt.Println()
}

// SubHeader prints a colored subheader.
func SubHeader(msg string, args ...interface{}) {
	fmt.Printf(Cyan+Bold+msg+Reset+"\n", args...)
}

// Dim prints dimmed text.
func DimText(msg string, args ...interface{}) {
	fmt.Printf(Dim+msg+Reset+"\n", args...)
}

// Label prints a labeled value.
func Label(label, value string) {
	fmt.Printf("  %s%-16s%s %s\n", Dim, label+":", Reset, value)
}

// StatusLine prints a status check line.
func StatusLine(ok bool, msg string) {
	if ok {
		fmt.Printf("  %s%s%s %s\n", Green, SymbolCheck, Reset, msg)
	} else {
		fmt.Printf("  %s%s%s %s\n", Red, SymbolCross, Reset, msg)
	}
}

// WarnLine prints a warning check line.
func WarnLine(msg string) {
	fmt.Printf("  %s%s%s %s\n", Yellow, SymbolWarn, Reset, msg)
}

// Confirm asks the user for a yes/no confirmation.
func Confirm(prompt string, defaultYes bool) bool {
	suffix := " [y/N]: "
	if defaultYes {
		suffix = " [Y/n]: "
	}
	fmt.Print(prompt + suffix)

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer == "" {
		return defaultYes
	}
	return answer == "y" || answer == "yes"
}

// Prompt asks the user for input with a default value.
func Prompt(label, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultValue
	}
	return answer
}

// SelectOne shows a numbered list and asks the user to select one.
func SelectOne(label string, options []string) (int, string) {
	fmt.Println(label)
	for i, opt := range options {
		fmt.Printf("  %s%d)%s %s\n", Cyan, i+1, Reset, opt)
	}
	fmt.Print("Select: ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return -1, ""
	}
	answer = strings.TrimSpace(answer)

	var idx int
	if _, err := fmt.Sscanf(answer, "%d", &idx); err != nil || idx < 1 || idx > len(options) {
		return -1, ""
	}
	return idx - 1, options[idx-1]
}

// Table prints a simple table.
func Table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Printf("  %-*s", widths[i]+2, h)
	}
	fmt.Println()

	// Print separator
	for i := range headers {
		fmt.Printf("  %s", strings.Repeat("─", widths[i]+1))
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("  %-*s", widths[i]+2, cell)
			}
		}
		fmt.Println()
	}
}
