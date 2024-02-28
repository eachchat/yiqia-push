package push

import "context"

// Message is the message to be pushed
type Message struct {
	DeviceTokens []string
	Payload      *Payload
}

// Payload is the payload of the message
type Payload struct {
	BusinessID    string
	Title         string
	Content       string
	CallBack      string
	CallbackParam string
}

// Push is the interface for push
type Push interface {
	// PushNotice pushes the message to the devices
	PushNotice(ctx context.Context, message *Message) error
}
