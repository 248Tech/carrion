package telnet

import "fmt"

// Command is a single telnet command to send.
type Command struct {
	Raw string
}

// SetGamePref builds a command string for setting a game pref (implementation-specific to 7DTD telnet).
func SetGamePref(name, value string) Command {
	return Command{Raw: fmt.Sprintf("setpref %s %s", name, value)}
}

// Say builds a command to send a chat message.
func Say(message string) Command {
	return Command{Raw: fmt.Sprintf("say %s", message)}
}

// Authenticate returns the login command (password).
func Authenticate(password string) Command {
	return Command{Raw: password}
}
