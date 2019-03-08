package controllers

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// Params maps an HTTP requests form, body, and query path data to the given
// model. The given model must be a pointer to a struct.
//
// Inspired by https://github.com/gorilla/schema.
func (c *Controller) Params(r *http.Request, model interface{}) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	// Assumptions:
	//	1. model is a struct.
	//	2. That struct is a pointer receiver.
	//
	// With these assumptions in place, iterate over each field, and try to
	// parse it from the request.
	rv := reflect.ValueOf(model)
	elem, err := getStructFromPtr(rv)
	if err != nil {
		return err
	}

	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)

		// If the field can't be set, any of the operations below will panic,
		// so we continue if that's the case.
		if !field.CanSet() {
			continue
		}

		key := elem.Type().Field(i).Name

		// Because they `key` here will always be a titlecased name, if this
		// value is an empty string, we check again with the lowercased name.
		value := r.FormValue(key)
		if value == "" {
			value = r.FormValue(strings.ToLower(key))
		}
		if value == "" {
			// At this point, we're out of options, so we assume the value is
			// unset.
			continue
		}

		// Set the field based on its kind. Its "kind" is its type.
		switch field.Kind() {
		case reflect.String:
			field.SetString(value)

		case reflect.Int, reflect.Int64:
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return errors.New(`field ` + key + ` expects type int, but "` + value + `" is not an int`)
			}
			field.SetInt(i)

		case reflect.Float64:
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return errors.New(`field ` + key + ` expects type float64, but "` + value + `" is not a float64`)
			}
			field.SetFloat(f)

		case reflect.Bool:
			b, err := strconv.ParseBool(value)
			if err != nil {
				return errors.New(`field ` + key + ` expects type bool, but "` + value + `" is not a bool`)
			}
			field.SetBool(b)

		default:
			// ok
		}
	}

	return nil
}

// getStructFromPtr checks to see if rv is a pointer to a struct. If it is, it
// returns its underlying value. If not, it returns the original value, along
// with an error message describing what the expected type for rv is, and what
// the actual type is.
func getStructFromPtr(rv reflect.Value) (reflect.Value, error) {
	kind := rv.Kind()
	if kind != reflect.Ptr {
		return rv, errors.New("expected interface to be a pointer to a struct, but got a " + kind.String())
	}

	elem := rv.Elem()
	underlyingKind := elem.Kind()
	if underlyingKind != reflect.Struct {
		return rv, errors.New("expected interface to be a pointer to a struct, but got a pointer to a " + underlyingKind.String())
	}

	return elem, nil
}
