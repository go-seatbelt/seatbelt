package seatbelt

import (
	_ "golang.org/x/text/message" // Required for commands to work.
)

// Data is the type for providing templates with data.
type Data map[string]interface{}

// A Flash is a map of flash messages.
type Flash map[string]string
