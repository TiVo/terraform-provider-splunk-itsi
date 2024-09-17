package util

import (
	"syscall"

	"golang.org/x/term"
)

// ReadPassword reads a password securely from the terminal.
func ReadPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}
