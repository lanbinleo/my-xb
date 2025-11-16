package main

import "fmt"

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func colorize(color, text string) string {
	return color + text + colorReset
}

func red(text string) string {
	return colorize(colorRed, text)
}

func green(text string) string {
	return colorize(colorGreen, text)
}

func yellow(text string) string {
	return colorize(colorYellow, text)
}

func blue(text string) string {
	return colorize(colorBlue, text)
}

func cyan(text string) string {
	return colorize(colorCyan, text)
}

func gray(text string) string {
	return colorize(colorGray, text)
}

func bold(text string) string {
	return colorize(colorBold, text)
}

func printSuccess(msg string) {
	fmt.Println(green("✓"), msg)
}

func printError(msg string) {
	fmt.Println(red("✗"), msg)
}

func printInfo(msg string) {
	fmt.Println(blue("ℹ"), msg)
}

func printWarning(msg string) {
	fmt.Println(yellow("⚠"), msg)
}

func printBanner(version string) {
	fmt.Println()
	fmt.Printf(" %s   %s\n",
		bold(cyan("▐▛███▜▌")),
		bold(cyan("MyXB"))+gray(" "+version))
	fmt.Printf("%s  %s\n",
		bold(cyan("▝▜█████▛▘")),
		gray("GPA Calculator & Score Tracker"))
	fmt.Printf("  %s    %s\n",
		bold(cyan("▘▘ ▝▝")),
		gray("Xiaobao Grade Viewer"))
	fmt.Println()
}
