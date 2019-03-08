package sessions_test

import (
	"reflect"
	"testing"

	"github.com/go-seatbelt/seatbelt/sessions"
)

func TestSessions(t *testing.T) {
	t.Parallel()

	var key string
	type testdata struct {
		ID   int64
		Name string
	}
	td := &testdata{ID: 1, Name: "test"}

	t.Run("save a session", func(t *testing.T) {
		k, err := sessions.Save(td)
		if err != nil {
			t.Fatalf("error saving session data: %+v", err)
		}
		key = k
	})

	t.Run("get a session", func(t *testing.T) {
		std := &testdata{}
		if err := sessions.Get(key, std); err != nil {
			t.Fatalf("error getting session data: %+v", err)
		}
		if !reflect.DeepEqual(std, td) {
			t.Fatalf("expected data to be equal but got %+v vs. %+v", td, std)
		}
	})

	t.Run("update a session", func(t *testing.T) {
		td.Name = "changed"
		if err := sessions.Put(key, td); err != nil {
			t.Fatalf("error updating session: %+v", err)
		}

		std := &testdata{}
		if err := sessions.Get(key, std); err != nil {
			t.Fatalf("error getting session data: %+v", err)
		}
		if !reflect.DeepEqual(std, td) {
			t.Fatalf("expected data to be equal but got %+v vs. %+v", td, std)
		}
	})

	t.Run("delete a session", func(t *testing.T) {
		if err := sessions.Delete(key); err != nil {
			t.Fatalf("error deleting session: %+v", err)
		}
	})

	t.Run("authorize a session, find it, then unauthorize it", func(t *testing.T) {
		td := &testdata{
			ID:   1,
			Name: "Ben",
		}

		token, err := sessions.Authorize(td)
		if err != nil {
			t.Fatalf("error authorizing td: %+v", err)
		}

		ftd := &testdata{}
		err = sessions.Find(token, ftd)
		if err != nil {
			t.Fatalf("error finding td: %+v", err)
		}
		if !reflect.DeepEqual(ftd, td) {
			t.Fatalf("expected tds to match, but got %+v vs %+v", ftd, td)
		}

		if err := sessions.Unauthorize(token); err != nil {
			t.Fatalf("error unauthorizing session: %+v", err)
		}
		if err := sessions.Find(token, ftd); err == nil {
			t.Fatalf("expected error")
		}
	})
}
