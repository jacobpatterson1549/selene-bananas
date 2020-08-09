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
	case reflect.String, reflect.Int, reflect.Int64:
		return src, nil
	case reflect.Int32: // TODO: rune hack - make tile.letter a string, make it public
		r := src.(int32)
		return string(r), nil
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

// toStruct converts the value to a map of field json names, to values.
func toStruct(v reflect.Value) (map[string]interface{}, error) {
	n := v.NumField()
	t := v.Type()
	m := make(map[string]interface{}, n)
	for i := 0; i < n; i++ {
		field := t.Field(i)
		jsonTags := field.Tag.Get("json")
		jsonName := strings.Split(jsonTags, ",")[0]
		if len(jsonName) == 0 {
			return nil, errors.New("field " + strconv.Itoa(i) + " of struct " + v.Type().Name() + "(" + field.Name + ") has no json name")
		}
		vi := v.Field(i)
		vii, err := toMap(vi.Interface())
		if err != nil {
			return nil, errors.New("getting value of field " + strconv.Itoa(i) + " of struct")
		}
		m[jsonName] = vii
	}
	return m, nil
}
