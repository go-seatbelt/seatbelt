package seatbelt

import (
	"os"
	"testing"
)

func TestOptions(t *testing.T) {
	o := &Option{}

	t.Run("a master.key file should be present after calling setDefaults", func(t *testing.T) {
		o.setDefaults()

		data, err := os.ReadFile("master.key")
		if err != nil {
			t.Fatalf("failed to read master.key file: %v", err)
		}
		if data == nil {
			t.Fatal("file is empty")
		}
	})
}
