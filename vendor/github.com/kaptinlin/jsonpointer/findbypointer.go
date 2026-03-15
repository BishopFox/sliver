package jsonpointer

import (
	"reflect"
	"strings"
)

// findByPointer optimized string-based find operation.
// Direct string parsing without path array allocation for better performance.
//
// TypeScript original code from findByPointer/v5.ts:
//
//	export const findByPointer = (pointer: string, val: unknown): Reference => {
//	  if (!pointer) return {val};
//	  let obj: Reference['obj'];
//	  let key: Reference['key'];
//	  let indexOfSlash = 0;
//	  let indexAfterSlash = 1;
//	  while (indexOfSlash > -1) {
//	    indexOfSlash = pointer.indexOf('/', indexAfterSlash);
//	    key = indexOfSlash > -1 ? pointer.substring(indexAfterSlash, indexOfSlash) : pointer.substring(indexAfterSlash);
//	    indexAfterSlash = indexOfSlash + 1;
//	    obj = val;
//	    if (isArray(obj)) {
//	      const length = obj.length;
//	      if (key === '-') key = length;
//	      else {
//	        const key2 = ~~key;
//	        if ('' + key2 !== key) throw new Error('INVALID_INDEX');
//	        key = key2;
//	        if (key < 0) throw 'INVALID_INDEX';
//	      }
//	      val = obj[key];
//	    } else if (typeof obj === 'object' && !!obj) {
//	      key = unescapeComponent(key);
//	      val = has(obj, key) ? (obj as any)[key] : undefined;
//	    } else throw 'NOT_FOUND';
//	  }
//	  return {val, obj, key};
//	};
func findByPointer(pointer string, val any) (*Reference, error) {
	if pointer == "" {
		return &Reference{Val: val}, nil
	}

	var obj any
	var key string
	indexOfSlash := 0
	indexAfterSlash := 1

	for indexOfSlash > -1 {
		indexOfSlash = strings.Index(pointer[indexAfterSlash:], "/")
		if indexOfSlash > -1 {
			indexOfSlash += indexAfterSlash
		}

		var keyStr string
		if indexOfSlash > -1 {
			keyStr = pointer[indexAfterSlash:indexOfSlash]
		} else {
			keyStr = pointer[indexAfterSlash:]
		}

		indexAfterSlash = indexOfSlash + 1
		obj = val

		switch {
		case isSliceOrArray(obj):
			arrayVal, err := derefValue(reflect.ValueOf(obj))
			if err != nil {
				return nil, err
			}

			index, err := validateAndAccessArray(keyStr, arrayVal.Len())
			if err != nil {
				return nil, err
			}
			val = arrayVal.Index(index).Interface()
			key = keyStr

		case isObjectPointer(obj) && obj != nil:
			keyStr = unescapeComponent(keyStr)
			key = keyStr

			objVal := reflect.ValueOf(obj)
			if objVal.Kind() == reflect.Map {
				mapKey := reflect.ValueOf(keyStr)
				mapVal := objVal.MapIndex(mapKey)
				if !mapVal.IsValid() {
					return nil, ErrKeyNotFound
				}
				val = mapVal.Interface()
			} else {
				if !structField(keyStr, &objVal) {
					return nil, ErrFieldNotFound
				}
				val = objVal.Interface()
			}

		default:
			return nil, ErrNotFound
		}
	}

	return &Reference{
		Val: val,
		Obj: obj,
		Key: key,
	}, nil
}

// isSliceOrArray checks if a value is a slice or array type after dereferencing pointers.
func isSliceOrArray(obj any) bool {
	if obj == nil {
		return false
	}
	objVal := reflect.ValueOf(obj)
	for objVal.Kind() == reflect.Pointer {
		if objVal.IsNil() {
			return false
		}
		objVal = objVal.Elem()
	}
	kind := objVal.Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

// isObjectPointer checks if a value is an object (map or struct) for pointer operations.
func isObjectPointer(val any) bool {
	if val == nil {
		return false
	}
	kind := reflect.TypeOf(val).Kind()
	return kind == reflect.Map || kind == reflect.Struct || kind == reflect.Pointer
}
