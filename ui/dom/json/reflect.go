package json

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// fromObject converts a json-styled object into the go interface.
func fromObject(dst, src interface{}) error {
	d := reflect.ValueOf(dst)
	if dk := d.Kind(); dk != reflect.Ptr {
		return errors.New("wanted dest kind to be a pointer, got was: " + dk.String())
	}
	de := d.Elem()
	dek := de.Kind()
	s := reflect.ValueOf(src)
	sk := s.Kind()
	switch {
	case sk == reflect.Map:
		if dek != reflect.Struct {
			return errors.New("want dest to be a struct when source is map, got " + dek.String())
		}
	case sk != dek:
		if sk != reflect.Int && dek != reflect.Int64 { // ints should be passed as int64s.
			return errors.New("want dest to be " + sk.String() + ", got " + dek.String())
		}
	}
	switch sk {
	case reflect.String:
		ss := s.String()
		de.SetString(ss)
	case reflect.Int, reflect.Int64:
		si := s.Int()
		de.SetInt(si)
	case reflect.Slice:
		return fromSlice(de, s)
	case reflect.Map:
		return fromStruct(de, s)
	default:
		return errors.New("unsupported source kind: " + sk.String())
	}
	return nil
}

// toObject converts the interface to a json-styled object that uses only strings, ints, slices, and maps of strings to other objects for structs.
func toObject(src interface{}) (interface{}, error) {
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

// fromObject converts a json-styled slice into the go slice.
func fromSlice(de, s reflect.Value) error {
	n := s.Len()
	if n == 0 {
		return nil
	}
	det := de.Type()
	dete := det.Elem()
	sliceType := reflect.SliceOf(dete)
	v := reflect.MakeSlice(sliceType, 0, n)
	for i := 0; i < n; i++ {
		si := s.Index(i)
		sii := si.Interface()
		vi := reflect.New(dete)
		vii := vi.Interface()
		if err := fromObject(vii, sii); err != nil {
			return errors.New("reading value from slice at index " + strconv.Itoa(i) + ": " + err.Error())
		}
		vie := vi.Elem()
		v = reflect.Append(v, vie)
	}
	de.Set(v)
	return nil
}

// toSlice converts the value to an interface array.
func toSlice(v reflect.Value) ([]interface{}, error) {
	n := v.Len()
	if n == 0 {
		return nil, nil
	}
	s := make([]interface{}, n)
	for i := 0; i < n; i++ {
		vi := v.Index(i)
		o, err := toObject(vi.Interface())
		if err != nil {
			return nil, errors.New("converting to slice at index " + strconv.Itoa(i) + ": " + err.Error())
		}
		s[i] = o
	}
	return s, nil
}

// FromStruct converts a json-styled map into the go struct.
func fromStruct(de, s reflect.Value) error {
	keyIndexes := structJSONTagIndexes(de)
	sk := s.MapKeys()
	for _, k := range sk {
		ks := k.String()
		i, ok := keyIndexes[ks]
		if !ok {
			continue
		}
		defi := de.Field(i)
		t := defi.Type()
		vi := reflect.New(t)
		vii := vi.Interface()
		sv := s.MapIndex(k)
		svi := sv.Interface()
		if err := fromObject(vii, svi); err != nil {
			return errors.New("adding value to named " + ks + " to struct at index " + strconv.Itoa(i) + ": " + err.Error())
		}
		vie := vi.Elem()
		defi.Set(vie)
	}
	return nil
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
		f := v.Field(i)
		o, err := toObject(f.Interface())
		if err != nil {
			return nil, errors.New("getting value of field " + strconv.Itoa(i) + " of struct")
		}
		if len(jsonTags) == 2 && jsonTags[1] == "omitempty" {
			vz := reflect.ValueOf(o)
			if vz.IsZero() {
				continue
			}
		}
		m[jsonName] = o
	}
	if len(m) == 0 {
		return nil, nil
	}
	return m, nil
}

var typeStructJSONTagIndexes = make(map[reflect.Type]map[string]int)

// structJSONTagIndexes builds a map of struct json tags to their indexs in the type's fields.
func structJSONTagIndexes(v reflect.Value) map[string]int {
	t := v.Type()
	if m, ok := typeStructJSONTagIndexes[t]; ok {
		return m
	}
	n := v.NumField()
	m := make(map[string]int, n)
	for i := 0; i < n; i++ {
		f := t.Field(i)
		jsonTag := f.Tag.Get("json")
		jsonTags := strings.Split(jsonTag, ",")
		jsonName := jsonTags[0]
		if jsonName != "" && jsonName != "-" {
			m[jsonName] = i
		}
	}
	typeStructJSONTagIndexes[t] = m
	return m
}
