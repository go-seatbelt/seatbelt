package migrations

import (
	"sort"
	"strings"
)

type DataType int

const (
	Text DataType = iota
	String
	BigInt
)

// A Table contains the columns, data types, and constraints of a SQL "CREATE
// TABLE" statement.
type Table struct {
	name string

	// The string to create the primary key.
	ID string

	// The columns of the create table statements.
	Columns []string

	// The timestamp table statements. These go last when creating table
	// statements.
	TimeColumns []string
}

func (t *Table) primaryKey() {
	// Set the field,
	//	id bigserial primary key,
	t.ID = "id bigserial primary key"
}

// BigInt creates column with given name with the data type bigint.
func (t *Table) BigInt(name string) {
	t.Columns = append(t.Columns, name+" bigint")
}

// Float creates column with given name with the data type float.
func (t *Table) Float(name string) {
	t.Columns = append(t.Columns, name+" float")
}

// String creates column with given name with the data type varchar(255).
func (t *Table) String(name string) {
	t.Columns = append(t.Columns, name+" varchar(255)")
}

// Text creates column with given name with the data type text.
func (t *Table) Text(name string) {
	t.Columns = append(t.Columns, name+" text")
}

// Timestamps creates two timestamptz columns: created_at and updated_at.
func (t *Table) Timestamps() {
	t.TimeColumns = append(t.TimeColumns, "created_at timestamptz default now()")
	t.TimeColumns = append(t.TimeColumns, "updated_at timestamptz default now()")
}

//
// --- Kinds of migrations. ---
//

func AddColumn(name, column string, dataType DataType) {

}

func AddIndex(name, column string) {}

// CreateTable creates a table with the given name.
func CreateTable(name string, fn func(t *Table)) string {
	t := &Table{
		name:        name,
		Columns:     make([]string, 0),
		TimeColumns: make([]string, 0),
	}
	t.primaryKey()

	if fn != nil {
		fn(t)
	}

	var builder strings.Builder
	builder.WriteString("CREATE TABLE")
	builder.WriteString(" ")
	builder.WriteString(t.name)
	builder.WriteString(" ")
	builder.WriteString("(")
	builder.WriteString("\n")

	// Write ID statements.
	builder.WriteString("    ")
	builder.WriteString(t.ID)
	if len(t.Columns) == 0 && len(t.TimeColumns) == 0 {
		builder.WriteString("\n")
	} else {
		builder.WriteString(",\n")
	}

	// Write the column statements.
	sort.Strings(t.Columns)
	for i, col := range t.Columns {
		builder.WriteString("    ")
		builder.WriteString(col)

		// If this isn't the final line, then write a comma.
		if len(t.Columns)-1 != i {
			builder.WriteString(",")
		} else if len(t.TimeColumns) != 0 {
			// If there are timestamps, then we need to write trailing commas.
			builder.WriteString(",")
		}
		builder.WriteString("\n")
	}

	// Write the timestamp columns.
	for i, col := range t.TimeColumns {
		builder.WriteString("    ")
		builder.WriteString(col)

		// If this isn't the final line, then write a comma.
		if len(t.TimeColumns)-1 != i {
			builder.WriteString(",")
		}
		builder.WriteString("\n")
	}

	builder.WriteString(");")

	return builder.String()
}

func RenameColumn(name, current, new string) {}
