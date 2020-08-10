package json

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// fromMap maps the source value into the destination value using reflection.
func fromMap(dst, src interface{}) error {
	return nil // TODO
}

// toMap converts the interface to a map if it is a slice and passes through primitive types.
func toMap(src interface{}) (interface{}, error) {
	if src == nil {
		return nil, nil
	}
	v := reflect.ValueOf(src)
	switch k := v.Kind(); k {
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int64:
		return v.Int(), nil // converts to int64
	case reflect.Slice:
		return toSlice(v)
	case reflect.Struct:
		return toStruct(v)
	default:
		return nil, errors.New("unsupported kind: " + k.String())
	}
}

// toSlice converts the value to an interface array.
func toSlice(v reflect.Value) ([]interface{}, error) {
	n := v.Len()
	if n == 0 {
		return nil, nil
	}
	s := make([]interface{}, n)
	for i := 0; i < n; i++ {
		vi, err := toMap(v.Index(i).Interface())
		if err != nil {
			return nil, errors.New("index " + strconv.Itoa(i) + ": " + err.Error())
		}
		s[i] = vi
	}
	return s, nil
}

// toStruct converts the value to a map of field json names to values.
// This only uses a subset of golang's json field name logic, always requiring json tags (never resorting to field names) and not allowing the '-' tag if tagged as "-,".
func toStruct(v reflect.Value) (map[string]interface{}, error) {
	n := v.NumField()
	t := v.Type()
	m := make(map[string]interface{}, n)
	for i := 0; i < n; i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		jsonTags := strings.Split(jsonTag, ",")
		jsonName := jsonTags[0]
		switch jsonName {
		case "-":
			continue
		case "":
			return nil, errors.New("field " + strconv.Itoa(i) + " of struct " + v.Type().Name() + "(" + field.Name + ") has no json name")
		}
		vi := v.Field(i)
		vii, err := toMap(vi.Interface())
		if err != nil {
			return nil, errors.New("getting value of field " + strconv.Itoa(i) + " of struct")
		}
		if len(jsonTags) == 2 && jsonTags[1] == "omitempty" {
			viii := reflect.ValueOf(vii)
			if viii.IsZero() {
				continue
			}
		}
		m[jsonName] = vii
	}
	if len(m) == 0 {
		return nil, nil
	}
	return m, nil
}
