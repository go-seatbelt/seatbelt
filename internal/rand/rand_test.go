package rand_test

import (
	"testing"

	"github.com/go-seatbelt/seatbelt/internal/rand"
)

func TestRandomString(t *testing.T) {
	t.Parallel()

	a := rand.NewString(12)
	b := rand.NewString(12)

	if a == b {
		t.Fatalf("expected random string to be different")
	}
}
