package ui

import (
	"fmt"
	"os"
	"strings"
)

// Puts prints a styled line to stdout.
func Puts(s string) {
	fmt.Println(s)
}

// Putsf prints a formatted styled line to stdout.
func Putsf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

// Warn prints a warning message.
func Warn(msg string) {
	fmt.Println(Warning.Render(IconWarn + msg))
}

// Err prints an error message.
func Err(msg string) {
	// Force color output for errors to ensure visibility
	styled := Error.Copy().Bold(true).Render(IconError + msg)
	fmt.Fprintln(os.Stderr, styled)
}

// Ok prints a success message.
func Ok(msg string) {
	fmt.Println(Success.Render(IconOk + msg))
}

// Inf prints an info message.
func Inf(msg string) {
	fmt.Println(Info.Render("  " + msg))
}

// Header prints a section header.
func Header(s string) {
	fmt.Println()
	fmt.Println(Title.Render(s))
	fmt.Println(Muted.Render(strings.Repeat("─", len(s)+2)))
}

// Tip prints a helpful tip.
func Tip(msg string) {
	fmt.Println()
	fmt.Println(Muted.Render("  tip: " + msg))
}

// Kv prints a key-value pair, padded.
func Kv(key string, value string) {
	k := KeyStyle.Render(fmt.Sprintf("  %-12s", key))
	v := ValueStyle.Render(value)
	fmt.Printf("%s %s\n", k, v)
}

// Greet prints a whimsical greeting based on context.
func Greet(name string) string {
	if name == "" {
		return IconMine + "Hey there!"
	}
	return fmt.Sprintf("%sHey %s!", IconMine, name)
}

// Die prints an error message and exits.
func Die(msg string) {
	Err(msg)
	os.Exit(1)
}

// Dief prints a formatted error message and exits.
func Dief(format string, args ...any) {
	Die(fmt.Sprintf(format, args...))
}
