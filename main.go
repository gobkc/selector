package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unicode"
	"unsafe"

	"github.com/gobkc/selector/internal"
	"golang.org/x/term"
	"log/slog"
)

var (
	purpleBackground = "\033[48;5;93m"
	whiteText        = "\033[97m"
	resetColor       = "\033[0m"
	parameter        internal.Parameter
	searchText       string
	input            = make(chan rune)
	oldState         *term.State
)

func main() {
	// Setup a deferred function to ensure the terminal state is restored on exit
	defer func() {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			if oldState != nil {
				term.Restore(int(os.Stdin.Fd()), oldState)
				fmt.Print("\033[?1049l") // Reset to normal buffer
			}
		}
	}()

	if err := internal.BindCmd(&parameter); err != nil {
		slog.Warn("failed to bind command parameter: ", slog.String("more", err.Error()))
		os.Exit(1)
	}
	printHelp(parameter.Help)

	options, _ := internal.ReadInput(parameter.Options)
	index := ""
	if len(options) > 0 {
		index = options[0]
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGWINCH, syscall.SIGINT)

	var err error
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}
		defer func() {
			term.Restore(int(os.Stdin.Fd()), oldState)
			fmt.Print("\033[?1049l") // Reset to normal buffer
		}()
	}

	go readInput(input)
	clearAndDraw(options, index)

	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGWINCH {
				clearAndDraw(options, index)
			} else if sig == syscall.SIGINT {
				if term.IsTerminal(int(os.Stdin.Fd())) {
					term.Restore(int(os.Stdin.Fd()), oldState)
				}
				fmt.Print("\033[2J\033[H") // Clear screen
				os.Exit(1)
			}
		case char := <-input:
			handleInput(char, options, &index)
			clearAndDraw(options, index)
		}
	}
}

func readInput(input chan<- rune) {
	reader := bufio.NewReader(os.Stdin)
	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			return
		}
		input <- char
	}
}

func handleInput(char rune, options []string, index *string) {
	switch char {
	case '\033': // ESC sequence
		nextChar := <-input
		if nextChar == '[' {
			finalChar := <-input
			switch finalChar {
			case 'A': // up
				*index = prevIndex(options, *index)
			case 'B': // down
				*index = nextIndex(options, *index)
			}
		}
	case 13: // Enter
		fmt.Print("\033[2J\033[H")
		term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Print("\033[?1049l") // Reset to normal buffer
		fmt.Println(*index)
		os.Exit(0)
	case 127: // Delete
		if len(searchText) > 0 {
			fmt.Print("\b \b")
			searchText = searchText[:len(searchText)-1]
		}
	default:
		if isValidInput(string(char)) {
			searchText += string(char)
			fmt.Print(string(char))
			searchVal := searchOptions(options, searchText)
			if searchVal != "" {
				*index = searchVal
			}
		}
	}
}

func getWindowSize() (int, int) {
	ws := &winsize{}
	_, _, _ = syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)
	return int(ws.Row), int(ws.Col)
}

func searchOptions(options []string, searchText string) string {
	searchList := strings.Split(searchText, ",")
	for _, option := range options {
		for _, item := range searchList {
			if strings.Contains(option, item) {
				return option
			}
		}
	}
	return ""
}

func clearAndDraw(options []string, index string) {
	_, cols := getWindowSize()
	if cols == 0 {
		cols = len(parameter.Title) + len(searchText) + 43
	}
	fmt.Print("\033[2J\033[H")
	fmt.Printf("%s: %s\n\r", parameter.Title, searchText)

	for _, opt := range options {
		if opt == index {
			fmt.Print(purpleBackground + whiteText)
		} else {
			fmt.Print(resetColor)
		}
		fmt.Printf("%-*s\n\r", cols, " "+opt)
	}

	fmt.Print(resetColor)
	fmt.Printf("\033[1;%dH", len(parameter.Title)+len(searchText)+3)
}

func prevIndex(options []string, currentIndex string) string {
	for i, opt := range options {
		if opt == currentIndex {
			if i > 0 {
				return options[i-1]
			}
			return options[len(options)-1]
		}
	}
	return currentIndex
}

func nextIndex(options []string, currentIndex string) string {
	for i, opt := range options {
		if opt == currentIndex {
			if i < len(options)-1 {
				return options[i+1]
			}
			return options[0]
		}
	}
	return currentIndex
}

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

func printHelp(help bool) {
	if help {
		fmt.Println("Usage:")
		fmt.Println(`selector -t="please input the right title" -o="aaa\nbbb"`)
		os.Exit(0)
	}
}

func isValidInput(input string) bool {
	for _, r := range input {
		if !(unicode.IsLetter(r) || unicode.IsSpace(r) || unicode.Is(unicode.Scripts["Han"], r) || unicode.IsNumber(r)) {
			return false
		}
	}
	return true
}
