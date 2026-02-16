package notify

import "fmt"

// Message represents a single log message with a level.
type Message struct {
	Level string
	Text  string
}

// MessageLog collects messages during execution.
type MessageLog struct {
	Messages []Message
	HasError bool
}

// Info appends an info-level message and prints it.
func (l *MessageLog) Info(text string) {
	fmt.Println(text)
	l.Messages = append(l.Messages, Message{Level: "info", Text: text})
}

// Error appends an error-level message and prints it.
func (l *MessageLog) Error(text string) {
	fmt.Println(text)
	l.Messages = append(l.Messages, Message{Level: "error", Text: text})
	l.HasError = true
}

// Notifier sends a collected message log somewhere.
type Notifier interface {
	Send(log *MessageLog) error
}
