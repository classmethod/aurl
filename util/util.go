package util

import (
	"errors"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/byteness/aws-vault/v7/prompt"
	"golang.org/x/term"
)

// promptWithDefault prompts the user with a message and returns the input or default value if empty
func TerminalPromptWithDefault(message, defaultValue string) (string, error) {
	value, err := TerminalPrompt(message)
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func TerminalPrompt(message string) (string, error) {
	if !lineEditable() {
		return prompt.TerminalPrompt(message)
	}
	return readTerminal(message, false)
}

func TerminalSecretPrompt(message string) (string, error) {
	if !lineEditable() {
		return prompt.TerminalSecretPrompt(message)
	}
	return readTerminal(message, true)
}

// Legacy Windows conhost does not interpret the ANSI escapes x/term emits; non-TTY stdio falls back to /dev/tty.
func lineEditable() bool {
	return runtime.GOOS != "windows" && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
}

func readTerminal(message string, secret bool) (string, error) {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, state)

	t := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stderr}, message)
	// x/term assumes 80 columns and hard-wraps there unless told the real size.
	if w, h, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
		t.SetSize(w, h)
	}
	var line string
	if secret {
		line, err = readMasked(message)
	} else {
		line, err = t.ReadLine()
	}
	if err == io.EOF {
		// x/term returns io.EOF for Ctrl-C and for Ctrl-D on an empty line without moving to a new line.
		os.Stderr.WriteString("\r\n")
		return "", errors.New("input cancelled")
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Echoes '*' per rune so the user can see that input was accepted; no cursor movement.
func readMasked(message string) (string, error) {
	os.Stderr.WriteString(message)
	var runes []rune
	buf := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}
		for _, r := range string(buf[:n]) {
			switch r {
			case '\r', '\n':
				os.Stderr.WriteString("\r\n")
				return string(runes), nil
			case 0x03, 0x04:
				return "", io.EOF
			case 0x7f, '\b':
				if len(runes) > 0 {
					runes = runes[:len(runes)-1]
					os.Stderr.WriteString("\b \b")
				}
			case 0x1b:
				// Drop the rest of the read: it is an escape sequence (arrow keys etc.)
				goto next
			default:
				if r >= ' ' {
					runes = append(runes, r)
					os.Stderr.WriteString("*")
				}
			}
		}
	next:
	}
}
