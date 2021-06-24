package gojsonq

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func abs(i int) int {
	if i < 0 {
		i = -1 * i
	}
	return i
}

func isIndex(in string) bool {
	return strings.HasPrefix(in, "[") && strings.HasSuffix(in, "]")
}

func getIndex(in string) (int, error) {
	if !isIndex(in) {
		return -1, fmt.Errorf("invalid index")
	}
	is := strings.TrimLeft(in, "[")
	is = strings.TrimRight(is, "]")
	oint, err := strconv.Atoi(is)
	if err != nil {
		return -1, err
	}
	return oint, nil
}

func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

// toFloat64 converts interface{} value to float64 if value is numeric else return false
func toFloat64(v interface{}) (float64, bool) {
	var f float64
	flag := true
	// as Go convert the json Numeric value to float64
	switch u := v.(type) {
	case int:
		f = float64(u)
	case int8:
		f = float64(u)
	case int16:
		f = float64(u)
	case int32:
		f = float64(u)
	case int64:
		f = float64(u)
	case float32:
		f = float64(u)
	case float64:
		f = u
	default:
		flag = false
	}
	return f, flag
}

// sortList sorts a list of interfaces
func sortList(list []interface{}, asc bool) []interface{} {
	var ss []string
	var ff []float64
	var result []interface{}
	for _, v := range list {
		// sort elements for string
		if sv, ok := v.(string); ok {
			ss = append(ss, sv)
		}
		// sort elements for float64
		if fv, ok := v.(float64); ok {
			ff = append(ff, fv)
		}
	}

	if len(ss) > 0 {
		if asc {
			sort.Strings(ss)
		} else {
			sort.Sort(sort.Reverse(sort.StringSlice(ss)))
		}
		for _, v := range ss {
			result = append(result, v)
		}
	}
	if len(ff) > 0 {
		if asc {
			sort.Float64s(ff)
		} else {
			sort.Sort(sort.Reverse(sort.Float64Slice(ff)))
		}
		for _, v := range ff {
			result = append(result, v)
		}
	}
	return result
}

type sortMap struct {
	data      interface{}
	key       string
	desc      bool
	separator string
	errs      []error
}

// Sort sorts the slice of maps
func (s *sortMap) Sort(data interface{}) {
	s.data = data
	sort.Sort(s)
}

// Len satisfies the sort.Interface
func (s *sortMap) Len() int {
	return reflect.ValueOf(s.data).Len()
}

// Swap satisfies the sort.Interface
func (s *sortMap) Swap(i, j int) {
	if i > j {
		i, j = j, i
	}
	list := reflect.ValueOf(s.data)
	tmp := list.Index(i).Interface()
	list.Index(i).Set(list.Index(j))
	list.Index(j).Set(reflect.ValueOf(tmp))
}

// TODO: need improvement
// Less satisfies the sort.Interface
// This will work for string/float64 only
func (s *sortMap) Less(i, j int) (res bool) {
	list := reflect.ValueOf(s.data)
	x := list.Index(i).Interface()
	y := list.Index(j).Interface()

	// compare nested values
	if strings.Contains(s.key, s.separator) {
		xv, errX := getNestedValue(x, s.key, s.separator)
		if errX != nil {
			s.errs = append(s.errs, errX)
		}
		yv, errY := getNestedValue(y, s.key, s.separator)
		if errY != nil {
			s.errs = append(s.errs, errY)
		}
		res = s.compare(xv, yv)
	}

	xv, okX := x.(map[string]interface{})
	if !okX {
		return
	}
	yv := y.(map[string]interface{})
	if mvx, ok := xv[s.key]; ok {
		mvy := yv[s.key]
		res = s.compare(mvx, mvy)
	}

	return
}

// compare compare two values
func (s *sortMap) compare(x, y interface{}) (res bool) {
	if mfv, ok := x.(float64); ok {
		if mvy, oky := y.(float64); oky {
			if s.desc {
				return mfv > mvy
			}
			res = mfv < mvy
		}
	}

	if mfv, ok := x.(string); ok {
		if mvy, oky := y.(string); oky {
			if s.desc {
				return mfv > mvy
			}
			res = mfv < mvy
		}
	}

	return
}

// getNestedValue fetch nested value from node
func getNestedValue(input interface{}, node, separator string) (interface{}, error) {
	pp := strings.Split(node, separator)
	for _, n := range pp {
		if isIndex(n) {
			// find slice/array
			if arr, ok := input.([]interface{}); ok {
				indx, err := getIndex(n)
				if err != nil {
					return input, err
				}
				arrLen := len(arr)
				if arrLen == 0 ||
					indx > arrLen-1 {
					return empty, errors.New("empty array")
				}
				input = arr[indx]
			}
		} else {
			// find in map
			validNode := false
			if mp, ok := input.(map[string]interface{}); ok {
				input, ok = mp[n]
				validNode = ok
			}

			// find in group data
			if mp, ok := input.(map[string][]interface{}); ok {
				input, ok = mp[n]
				validNode = ok
			}

			if !validNode {
				return empty, fmt.Errorf("invalid node name %s", n)
			}
		}
	}

	return input, nil
}

// makeAlias provide syntactic suger. when provide Property name as "user.name as userName"
// it return userName as output and pure node name like: "user.name".
// If "user.name" does not use "as" clause then it'll return "user.name", "user.name"
func makeAlias(in, separator string) (string, string) {
	const alias = " as "
	in = strings.Replace(in, " As ", alias, -1)
	in = strings.Replace(in, " AS ", alias, -1)

	if strings.Contains(in, alias) {
		ss := strings.Split(in, alias)
		return strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])
	}

	if strings.Contains(in, separator) {
		ss := strings.Split(in, separator)
		return in, ss[len(ss)-1]
	}

	return in, in
}

// length return length of strings/array/map
func length(v interface{}) (int, error) {
	switch val := v.(type) {
	case string:
		return len(val), nil
	case []interface{}:
		return len(val), nil
	case map[string]interface{}:
		return len(val), nil
	default:
		return -1, errors.New("invalid type for length")
	}
}
