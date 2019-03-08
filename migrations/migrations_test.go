package migrations_test

import (
	"testing"

	"github.com/go-seatbelt/seatbelt/migrations"
)

func TestMigrations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		expectedSQL  string
		generatedSQL string
	}{
		{
			name: "typical create statement test",
			expectedSQL: `CREATE TABLE test (
    id bigserial primary key,
    balance float,
    body text,
    text varchar(255),
    created_at timestamptz default now(),
    updated_at timestamptz default now()
);`,
			generatedSQL: migrations.CreateTable("test", func(t *migrations.Table) {
				t.Float("balance")
				t.String("text")
				t.Text("body")
				t.Timestamps()
			}),
		},

		{
			name: "create a table with only an ID",
			expectedSQL: `CREATE TABLE test (
    id bigserial primary key
);`,
			generatedSQL: migrations.CreateTable("test", nil),
		},

		{
			name: "create a table without timestamps",
			expectedSQL: `CREATE TABLE test (
    id bigserial primary key,
    balance float,
    body text,
    text varchar(255)
);`,
			generatedSQL: migrations.CreateTable("test", func(t *migrations.Table) {
				t.Float("balance")
				t.String("text")
				t.Text("body")
			}),
		},

		{
			name: "create a table with only a primary key and timestamps",
			expectedSQL: `CREATE TABLE test (
    id bigserial primary key,
    created_at timestamptz default now(),
    updated_at timestamptz default now()
);`,
			generatedSQL: migrations.CreateTable("test", func(t *migrations.Table) {
				t.Timestamps()
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedSQL != tc.generatedSQL {
				t.Fatalf("expected:\n\n%s\n\nbut got:\n\n%s\n\n", tc.expectedSQL, tc.generatedSQL)
			}
		})
	}
}
