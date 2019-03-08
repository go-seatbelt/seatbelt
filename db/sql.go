package db

import (
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/go-seatbelt/seatbelt/internal/config"
)

// init creates a new database connection using the config values provided in
// config/application.yml.
func init() {
	options := &pg.Options{
		User:     config.Username,
		Database: config.Database,
		Password: config.Password,
	}

	db := pg.Connect(options)

	if !config.IsTest {
		Migrate("up", db)
	}

	DB = db
}

// DB is a global connection to a Postgres database.
var DB *pg.DB

// DefaultFilter is the default filter for SQL queries. It applies query
// filters based on the URL query parameters containeed in the url.Values
// field.
//
// It applies pagination, and handles ordering results. By default, the sort
// direction is ASC. If the sort parameter starts with a "-", then the sort
// direction is DESC. This is specified by the JSON API spec.
var DefaultFilter = func(v url.Values) func(*orm.Query) (*orm.Query, error) {
	return func(q *orm.Query) (*orm.Query, error) {
		if limit, err := strconv.Atoi(v.Get("limit")); err == nil {
			q = q.Limit(limit)
		} else {
			q.Limit(20)
		}
		if offset, err := strconv.Atoi(v.Get("offset")); err == nil {
			q = q.Offset(offset)
		}
		if sort := v.Get("sort"); len(sort) != 0 {
			if sort[0] == '-' {
				sort = sort[1:] + " DESC"
			}
			q.Order(sort)
		}
		return q, nil
	}
}

// All returns all models, paginated by the values in v.
func All(model interface{}, v url.Values, relations ...string) error {
	q := DB.Model(&model).Apply(DefaultFilter(v))

	for _, relation := range relations {
		q.Relation(relation)
	}

	return q.Select()
}

// Find finds the given model by its primary key.
func Find(model interface{}, relations ...string) error {
	q := DB.Model(model).WherePK()

	for _, relation := range relations {
		q.Relation(relation)
	}

	return q.Select()
}

// FindBy returns the model that matches the given column and value. For
// example, if you wanted to find a user by email, you would write,
//
//	sql.FindBy(user, "email", "test@example.com")
//
// This function should only be used to find models by columns that have a
// `unique` constraint. In the case of the column and value matching
// multiple models, only the first result will be returned.
func FindBy(model interface{}, column string, value interface{}, relations ...string) error {
	q := DB.Model(model).Where(column+" = ?", value)

	for _, relation := range relations {
		q.Relation(relation)
	}

	return q.First()
}

// Save saves the given model into the database.
func Save(model interface{}) error {
	setTimestamp(model, "CreatedAt", "UpdatedAt")
	_, err := DB.Model(model).Returning("*").Insert()
	return err
}

// Update updates the given model for all fields that are not null.
func Update(model interface{}) error {
	setTimestamp(model, "UpdatedAt")
	_, err := DB.Model(model).WherePK().Returning("*").UpdateNotNull()
	return err
}

// Destroy permanently deletes the model by its ID.
func Destroy(model interface{}, id int64) error {
	_, err := DB.Model(model).Where("id = ?", id).Delete()
	return err
}

// setTimestamp attempts to set the current time on the model's given field.
//
// For example, to set a model's "UpdatedAt" field, you'd call,
//
//	`setTimestamp(model, "UpdatedAt")`
func setTimestamp(model interface{}, fields ...string) {
	el := reflect.ValueOf(model).Elem()
	if !el.IsValid() {
		return
	}

	for _, field := range fields {
		f := el.FieldByName(field)
		if !f.CanSet() {
			return
		}

		f.Set(reflect.ValueOf(time.Now()))
	}
}
