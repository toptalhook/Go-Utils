package utils

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/Laisky/errors/v2"
	"golang.org/x/term"
)

// InputPassword reads password from stdin input
// and returns it as a string.
func InputPassword(hint string, validator func(string) bool) (passwd string, err error) {
	fmt.Printf("%s: ", hint)

	for {
		bytepw, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return "", errors.Wrap(err, "read input password")
		}

		if validator != nil && !validator(string(bytepw)) {
			fmt.Printf("password is invalid, try again: ")
			continue
		}

		return string(bytepw), nil
	}
}

// InputYes require user input `y` or `Y` to continue
func InputYes(hint string) (ok bool, err error) {
	fmt.Printf("%s, input y/Y to continue: ", hint)

	var confirm string
	_, err = fmt.Scanln(&confirm)
	if err != nil {
		return ok, errors.Wrap(err, "read input")
	}

	if strings.ToLower(confirm) != "y" {
		return false, nil
	}

	return true, nil
}
