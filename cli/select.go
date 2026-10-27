package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

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
	filter := ""
	visible := items
	drawn := 0
	offset := 0
	draw := func() {
		if drawn > 0 {
			fmt.Fprintf(os.Stderr, "\x1b[%dA", drawn)
		}
		rows, width := len(visible), 0
		if w, h, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
			// Cursor-up stops at the top row and wrapped lines take extra rows, so keep the frame within the screen.
			rows, width = max(h-2, 1), w-1
		}
		offset = scrollOffset(cur, offset, len(visible), rows)
		end := min(offset+rows, len(visible))
		header := title + " " + filter
		if end-offset < len(visible) {
			header += fmt.Sprintf("  (%d/%d)", cur+1, len(visible))
		}
		// Raw mode disables output post-processing, so \n alone does not return the carriage.
		fmt.Fprintf(os.Stderr, "\x1b[2K%s\r\n", truncateRunes(header, width))
		for i := offset; i < end; i++ {
			prefix := "  "
			if i == cur {
				prefix = "> "
			}
			fmt.Fprintf(os.Stderr, "\x1b[2K%s\r\n", truncateRunes(prefix+visible[i], width))
		}
		drawn = 1 + end - offset
		if len(visible) == 0 {
			fmt.Fprint(os.Stderr, "\x1b[2K  (no matches)\r\n")
			drawn++
		}
		fmt.Fprint(os.Stderr, "\x1b[J")
	}
	draw()

	buf := make([]byte, 8)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}
		idx, newVisible, newFilter, done, cancel := applyInput(cur, items, visible, filter, buf[:n])
		if cancel {
			return "", errors.New("cancelled")
		}
		if done {
			return newVisible[idx], nil
		}
		if idx != cur || newFilter != filter {
			cur, visible, filter = idx, newVisible, newFilter
			draw()
		}
	}
}

// applyInput feeds one read to handleKey, splitting non-escape input byte by byte so pasted or fast-typed text is not dropped.
func applyInput(cur int, items, visible []string, filter string, input []byte) (int, []string, string, bool, bool) {
	for len(input) > 0 {
		key := input
		if key[0] != 0x1b {
			key = key[:1]
		}
		input = input[len(key):]
		idx, newFilter, done, cancel := handleKey(cur, len(visible), filter, key)
		if done || cancel {
			return cur, visible, filter, done, cancel
		}
		cur = idx
		if newFilter != filter {
			filter = newFilter
			visible = filterItems(items, filter)
		}
	}
	return cur, visible, filter, false, false
}

func scrollOffset(cur, offset, n, rows int) int {
	if n <= rows {
		return 0
	}
	if cur < offset {
		offset = cur
	}
	if cur >= offset+rows {
		offset = cur - rows + 1
	}
	return min(offset, n-rows)
}

// truncateRunes cuts s to at most width runes; width <= 0 means no limit.
func truncateRunes(s string, width int) string {
	r := []rune(s)
	if width <= 0 || len(r) <= width {
		return s
	}
	return string(r[:width])
}

func filterItems(items []string, filter string) []string {
	f := strings.ToLower(filter)
	var out []string
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), f) {
			out = append(out, item)
		}
	}
	return out
}

func handleKey(cur, n int, filter string, key []byte) (idx int, newFilter string, done, cancel bool) {
	switch string(key) {
	case "\x1b[A", "\x1bOA", "\x10":
		if n == 0 {
			return 0, filter, false, false
		}
		return (cur - 1 + n) % n, filter, false, false
	case "\x1b[B", "\x1bOB", "\x0e":
		if n == 0 {
			return 0, filter, false, false
		}
		return (cur + 1) % n, filter, false, false
	case "\r", "\n":
		return cur, filter, n > 0, false
	// A lone Esc is not a cancel key: slow links can split "\x1b[A" across reads
	case "\x03", "\x04":
		return cur, filter, false, true
	case "\x7f", "\x08":
		r := []rune(filter)
		if len(r) == 0 {
			return cur, filter, false, false
		}
		return 0, string(r[:len(r)-1]), false, false
	}
	if len(key) == 1 && key[0] >= 0x20 && key[0] <= 0x7e {
		return 0, filter + string(key), false, false
	}
	return cur, filter, false, false
}
