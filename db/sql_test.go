package db_test

import (
	"testing"

	"github.com/go-seatbelt/seatbelt/db"
)

func TestSQL(t *testing.T) {
	t.Parallel()

	if _, err := db.DB.ExecOne(`SELECT 1`); err != nil {
		t.Fatalf("error: %+v", err)
	}
}
