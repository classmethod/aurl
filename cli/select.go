package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/classmethod/aurl/util"
	"golang.org/x/term"
)

func selectFromList(title string, items []string) (string, error) {
	fd := int(os.Stdin.Fd())
	// Legacy Windows consoles do not interpret ANSI output escapes
	if runtime.GOOS == "windows" || !term.IsTerminal(fd) || !term.IsTerminal(int(os.Stderr.Fd())) {
		fmt.Fprintln(os.Stderr, title)
		for i, item := range items {
			fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, item)
		}
		answer, err := util.TerminalPrompt("Number: ")
		if err != nil {
			return "", err
		}
		return parseProfileChoice(items, answer)
	}

	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, state)
	fmt.Fprint(os.Stderr, "\x1b[?25l")
	defer fmt.Fprint(os.Stderr, "\x1b[?25h")

	cur := 0
	draw := func() {
		for i, item := range items {
			prefix := "  "
			if i == cur {
				prefix = "> "
			}
			// Raw mode disables output post-processing, so \n alone does not return the carriage.
			fmt.Fprintf(os.Stderr, "\x1b[2K%s%s\r\n", prefix, item)
		}
	}
	fmt.Fprintf(os.Stderr, "%s\r\n", title)
	draw()

	buf := make([]byte, 8)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}
		idx, done, cancel := nextIndex(cur, len(items), buf[:n])
		if cancel {
			return "", errors.New("cancelled")
		}
		if done {
			return items[cur], nil
		}
		if idx != cur {
			cur = idx
			fmt.Fprintf(os.Stderr, "\x1b[%dA", len(items))
			draw()
		}
	}
}

func nextIndex(cur, n int, key []byte) (idx int, done bool, cancel bool) {
	switch string(key) {
	case "\x1b[A", "\x1bOA", "k":
		return (cur - 1 + n) % n, false, false
	case "\x1b[B", "\x1bOB", "j":
		return (cur + 1) % n, false, false
	case "\r", "\n":
		return cur, true, false
	// A lone Esc is not a cancel key: slow links can split "\x1b[A" across reads
	case "q", "\x03", "\x04":
		return cur, false, true
	}
	return cur, false, false
}
