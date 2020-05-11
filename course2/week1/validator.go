package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

const tagName = "apivalidator"

type Validator interface {
	Validate(string, interface{}) error
}

// DefaultValidator does not perform any validations.
type DefaultValidator struct {
}

func (v DefaultValidator) Validate(name string, val interface{}) error {
	return nil
}

type StringValidator struct {
	required     bool
	min          *int
	max          *int
	enum         []string
	defaultValue *string
}

func (s StringValidator) Validate(name string, val interface{}) error {
	name = strings.ToLower(name)
	v := val.(string)
	if s.required && v == "" {
		return fmt.Errorf("%s must me not empty", name)
	}
	if s.min != nil {
		if utf8.RuneCountInString(v) < *s.min {
			return fmt.Errorf("%s len must be >= %d", name, *s.min)
		}
	}
	if s.max != nil {
		if utf8.RuneCountInString(v) > *s.max {
			return fmt.Errorf("%s len must be <= %d", name, *s.max)
		}
	}

	if s.defaultValue != nil && v == "" {
		v = *s.defaultValue
	}

	if len(s.enum) > 0 {
		isFound := false
		for i := range s.enum {
			if v == s.enum[i] {
				isFound = true
				break
			}
		}
		if !isFound {
			return fmt.Errorf("%s must be one of %s", name, "["+strings.Join(s.enum, ", ")+"]")
		}
	}
	return nil
}

type NumberValidator struct {
	min *int
	max *int
}

func (n NumberValidator) Validate(name string, val interface{}) error {
	name = strings.ToLower(name)
	v, _ := strconv.Atoi(fmt.Sprintf("%v", val))
	if n.min != nil {
		if v < *n.min {
			return fmt.Errorf("%s must be >= %d", name, *n.min)
		}
	}
	if n.max != nil {
		if v > *n.max {
			return fmt.Errorf("%s must be <= %d", name, *n.max)
		}
	}
	return nil
}

func validateStruct(s interface{}) error {

	// ValueOf returns a Value representing the run-time data
	v := reflect.ValueOf(s)

	for i := 0; i < v.NumField(); i++ {
		// Get the field tag value
		tag := v.Type().Field(i).Tag.Get(tagName)

		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}

		// Get a validator that corresponds to a tag
		validator := getValidatorFromFieldType(v.Field(i).Kind(), tag)

		// Perform validation
		err := validator.Validate(v.Type().Field(i).Name, v.Field(i).Interface())

		// Append error to results
		if err != nil {
			return err
			//errs = append(errs, fmt.Errorf("%s %s", v.Type().Field(i).Name, err.Error()))
		}
	}
	return nil
}

func getValidatorFromFieldType(k reflect.Kind, tag string) Validator {
	tagList := strings.Split(tag, ",")
	var v Validator
	switch k {
	case reflect.String:
		sv := StringValidator{}
		for _, tItem := range tagList {
			if tItem == "required" {
				sv.required = true
			}
			if strings.HasPrefix(tItem, "min=") {
				if m, err := strconv.Atoi(strings.TrimPrefix(tItem, "min=")); err == nil {
					sv.min = &m
				}
			}
			if strings.HasPrefix(tItem, "max=") {
				if m, err := strconv.Atoi(strings.TrimPrefix(tItem, "max=")); err == nil {
					sv.min = &m
				}
			}
			if strings.HasPrefix(tItem, "enum=") {
				e := strings.Split(strings.TrimPrefix(tItem, "enum="), "|")
				sv.enum = e
			}
			if strings.HasPrefix(tItem, "default=") {
				d := strings.TrimPrefix(tItem, "default=")
				sv.defaultValue = &d
			}
		}
		v = sv
	case reflect.Int:
		nv := NumberValidator{}
		for _, tItem := range tagList {
			if strings.HasPrefix(tItem, "min=") {
				if m, err := strconv.Atoi(strings.TrimPrefix(tItem, "min=")); err == nil {
					nv.min = &m
				}
			}
			if strings.HasPrefix(tItem, "max=") {
				if m, err := strconv.Atoi(strings.TrimPrefix(tItem, "max=")); err == nil {
					nv.max = &m
				}
			}
		}
		v = nv
	default:
		v = DefaultValidator{}
	}
	return v
}

func PopulateStruct(value interface{}, data url.Values) error {
	//Разбираем data

	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr {
		return nil
	}
	v = v.Elem()
	for i := 0; i < v.NumField(); i++ {
		// Get the field tag value
		tag := v.Type().Field(i).Tag.Get(tagName)
		tagList := strings.Split(tag, ",")
		// Ищем название структуры и соответствия названию из тега
		curFieldName := strings.ToLower(v.Type().Field(i).Name)
		for j := range tagList {
			if strings.HasPrefix(tagList[j], "paramname=") {
				curFieldName = strings.ToLower(strings.TrimPrefix(tagList[j], "paramname="))
				break
			}
		}
		switch v.Field(i).Kind() {
		case reflect.String:
			v.Field(i).SetString(data.Get(curFieldName))
		case reflect.Int:
			num, err := strconv.Atoi(data.Get(curFieldName))
			if err != nil {
				return fmt.Errorf("%s must be int", curFieldName)
			}
			v.Field(i).SetInt(int64(num))
		}
	}
	return nil
}

type responseData struct {
	Error    string          `json:"error"`
	Response json.RawMessage `json:"response,omitempty"`
}

func (r *responseData) Create() []byte {
	bs, _ := json.Marshal(r)
	return bs
}

func AuthorizeWrappers(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth") != "100500" {
			w.WriteHeader(http.StatusForbidden)
			rd := responseData{
				Error: "unauthorized",
			}
			w.Write(rd.Create())
			return
		}
		h.ServeHTTP(w, r)
	})
}
