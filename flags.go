package main

import (
	"fmt"
	"strings"
)

// authVar implements the Flag.Value interface and allows the user to specify
// authentication in the username[:password] form.
type authVar Auth

// Set sets the value for this authentication.
func (a *authVar) Set(value string) error {
	a.Enabled = true

	if strings.Contains(value, ":") {
		split := strings.SplitN(value, ":", 2)
		a.Username = split[0]
		a.Password = split[1]
	} else {
		a.Username = value
	}

	return nil
}

// String returns the string representation of this authentication.
func (a *authVar) String() string {
	if a.Password == "" {
		return a.Username
	}

	return fmt.Sprintf("%s:%s", a.Username, a.Password)
}
