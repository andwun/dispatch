// Copyright © 2014 Steve Francia <spf@spf13.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package cast

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"strconv"
	"strings"
	"time"

	jww "github.com/khlieng/name_pending/Godeps/_workspace/src/github.com/spf13/jwalterweatherman"
)

func ToTimeE(i interface{}) (tim time.Time, err error) {
	i = indirect(i)
	jww.DEBUG.Println("ToTimeE called on type:", reflect.TypeOf(i))

	switch s := i.(type) {
	case time.Time:
		return s, nil
	case string:
		d, e := StringToDate(s)
		if e == nil {
			return d, nil
		}
		return time.Time{}, fmt.Errorf("Could not parse Date/Time format: %v\n", e)
	default:
		return time.Time{}, fmt.Errorf("Unable to Cast %#v to Time\n", i)
	}
}

func ToDurationE(i interface{}) (d time.Duration, err error) {
	i = indirect(i)
	jww.DEBUG.Println("ToDurationE called on type:", reflect.TypeOf(i))

	switch s := i.(type) {
	case time.Duration:
		return s, nil
	case string:
		d, err = time.ParseDuration(s)
		return
	default:
		err = fmt.Errorf("Unable to Cast %#v to Duration\n", i)
		return
	}
}

func ToBoolE(i interface{}) (bool, error) {
	i = indirect(i)
	jww.DEBUG.Println("ToBoolE called on type:", reflect.TypeOf(i))

	switch b := i.(type) {
	case bool:
		return b, nil
	case nil:
		return false, nil
	case int:
		if i.(int) != 0 {
			return true, nil
		}
		return false, nil
	case string:
		return strconv.ParseBool(i.(string))
	default:
		return false, fmt.Errorf("Unable to Cast %#v to bool", i)
	}
}

func ToFloat64E(i interface{}) (float64, error) {
	i = indirect(i)
	jww.DEBUG.Println("ToFloat64E called on type:", reflect.TypeOf(i))

	switch s := i.(type) {
	case float64:
		return s, nil
	case float32:
		return float64(s), nil
	case int64:
		return float64(s), nil
	case int32:
		return float64(s), nil
	case int16:
		return float64(s), nil
	case int8:
		return float64(s), nil
	case int:
		return float64(s), nil
	case string:
		v, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return float64(v), nil
		} else {
			return 0.0, fmt.Errorf("Unable to Cast %#v to float", i)
		}
	default:
		return 0.0, fmt.Errorf("Unable to Cast %#v to float", i)
	}
}

func ToIntE(i interface{}) (int, error) {
	i = indirect(i)
	jww.DEBUG.Println("ToIntE called on type:", reflect.TypeOf(i))

	switch s := i.(type) {
	case int:
		return s, nil
	case int64:
		return int(s), nil
	case int32:
		return int(s), nil
	case int16:
		return int(s), nil
	case int8:
		return int(s), nil
	case string:
		v, err := strconv.ParseInt(s, 0, 0)
		if err == nil {
			return int(v), nil
		} else {
			return 0, fmt.Errorf("Unable to Cast %#v to int", i)
		}
	case float64:
		return int(s), nil
	case bool:
		if bool(s) {
			return 1, nil
		} else {
			return 0, nil
		}
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("Unable to Cast %#v to int", i)
	}
}

// From html/template/content.go
// Copyright 2011 The Go Authors. All rights reserved.
// indirect returns the value, after dereferencing as many times
// as necessary to reach the base type (or nil).
func indirect(a interface{}) interface{} {
	if a == nil {
		return nil
	}
	if t := reflect.TypeOf(a); t.Kind() != reflect.Ptr {
		// Avoid creating a reflect.Value if it's not a pointer.
		return a
	}
	v := reflect.ValueOf(a)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
}

// From html/template/content.go
// Copyright 2011 The Go Authors. All rights reserved.
// indirectToStringerOrError returns the value, after dereferencing as many times
// as necessary to reach the base type (or nil) or an implementation of fmt.Stringer
// or error,
func indirectToStringerOrError(a interface{}) interface{} {
	if a == nil {
		return nil
	}

	var errorType = reflect.TypeOf((*error)(nil)).Elem()
	var fmtStringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()

	v := reflect.ValueOf(a)
	for !v.Type().Implements(fmtStringerType) && !v.Type().Implements(errorType) && v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
}

func ToStringE(i interface{}) (string, error) {
	i = indirectToStringerOrError(i)
	jww.DEBUG.Println("ToStringE called on type:", reflect.TypeOf(i))

	switch s := i.(type) {
	case string:
		return s, nil
	case float64:
		return strconv.FormatFloat(i.(float64), 'f', -1, 64), nil
	case int:
		return strconv.FormatInt(int64(i.(int)), 10), nil
	case []byte:
		return string(s), nil
	case template.HTML:
		return string(s), nil
	case nil:
		return "", nil
	case fmt.Stringer:
		return s.String(), nil
	case error:
		return s.Error(), nil
	default:
		return "", fmt.Errorf("Unable to Cast %#v to string", i)
	}
}

func ToStringMapStringE(i interface{}) (map[string]string, error) {
	jww.DEBUG.Println("ToStringMapStringE called on type:", reflect.TypeOf(i))

	var m = map[string]string{}

	switch v := i.(type) {
	case map[interface{}]interface{}:
		for k, val := range v {
			m[ToString(k)] = ToString(val)
		}
		return m, nil
	case map[string]interface{}:
		for k, val := range v {
			m[ToString(k)] = ToString(val)
		}
		return m, nil
	case map[string]string:
		return v, nil
	default:
		return m, fmt.Errorf("Unable to Cast %#v to map[string]string", i)
	}
	return m, fmt.Errorf("Unable to Cast %#v to map[string]string", i)
}

func ToStringMapBoolE(i interface{}) (map[string]bool, error) {
	jww.DEBUG.Println("ToStringMapBoolE called on type:", reflect.TypeOf(i))

	var m = map[string]bool{}

	switch v := i.(type) {
	case map[interface{}]interface{}:
		for k, val := range v {
			m[ToString(k)] = ToBool(val)
		}
		return m, nil
	case map[string]interface{}:
		for k, val := range v {
			m[ToString(k)] = ToBool(val)
		}
		return m, nil
	case map[string]bool:
		return v, nil
	default:
		return m, fmt.Errorf("Unable to Cast %#v to map[string]bool", i)
	}
	return m, fmt.Errorf("Unable to Cast %#v to map[string]bool", i)
}

func ToStringMapE(i interface{}) (map[string]interface{}, error) {
	jww.DEBUG.Println("ToStringMapE called on type:", reflect.TypeOf(i))

	var m = map[string]interface{}{}

	switch v := i.(type) {
	case map[interface{}]interface{}:
		for k, val := range v {
			m[ToString(k)] = val
		}
		return m, nil
	case map[string]interface{}:
		return v, nil
	default:
		return m, fmt.Errorf("Unable to Cast %#v to map[string]interface{}", i)
	}

	return m, fmt.Errorf("Unable to Cast %#v to map[string]interface{}", i)
}

func ToSliceE(i interface{}) ([]interface{}, error) {
	jww.DEBUG.Println("ToSliceE called on type:", reflect.TypeOf(i))

	var s []interface{}

	switch v := i.(type) {
	case []interface{}:
		for _, u := range v {
			s = append(s, u)
		}
		return s, nil
	case []map[string]interface{}:
		for _, u := range v {
			s = append(s, u)
		}
		return s, nil
	default:
		return s, fmt.Errorf("Unable to Cast %#v of type %v to []interface{}", i, reflect.TypeOf(i))
	}

	return s, fmt.Errorf("Unable to Cast %#v to []interface{}", i)
}

func ToStringSliceE(i interface{}) ([]string, error) {
	jww.DEBUG.Println("ToStringSliceE called on type:", reflect.TypeOf(i))

	var a []string

	switch v := i.(type) {
	case []interface{}:
		for _, u := range v {
			a = append(a, ToString(u))
		}
		return a, nil
	case []string:
		return v, nil
	case string:
		return strings.Fields(v), nil
	default:
		return a, fmt.Errorf("Unable to Cast %#v to []string", i)
	}

	return a, fmt.Errorf("Unable to Cast %#v to []string", i)
}

func ToIntSliceE(i interface{}) ([]int, error) {
	jww.DEBUG.Println("ToIntSliceE called on type:", reflect.TypeOf(i))

	if i == nil {
		return []int{}, fmt.Errorf("Unable to Cast %#v to []int", i)
	}

	switch v := i.(type) {
	case []int:
		return v, nil
	}

	kind := reflect.TypeOf(i).Kind()
	switch kind {
	case reflect.Slice, reflect.Array:
		s := reflect.ValueOf(i)
		a := make([]int, s.Len())
		for j := 0; j < s.Len(); j++ {
			val, err := ToIntE(s.Index(j).Interface())
			if err != nil {
				return []int{}, fmt.Errorf("Unable to Cast %#v to []int", i)
			}
			a[j] = val
		}
		return a, nil
	default:
		return []int{}, fmt.Errorf("Unable to Cast %#v to []int", i)
	}

	return []int{}, fmt.Errorf("Unable to Cast %#v to []int", i)
}

func StringToDate(s string) (time.Time, error) {
	return parseDateWith(s, []string{
		time.RFC3339,
		"2006-01-02T15:04:05", // iso8601 without timezone
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		"2006-01-02 15:04:05Z07:00",
		"02 Jan 06 15:04 MST",
		"2006-01-02",
		"02 Jan 2006",
	})
}

func parseDateWith(s string, dates []string) (d time.Time, e error) {
	for _, dateType := range dates {
		if d, e = time.Parse(dateType, s); e == nil {
			return
		}
	}
	return d, errors.New(fmt.Sprintf("Unable to parse date: %s", s))
}
