package plivo

import (
	"log"
	"reflect"
	"strconv"
	"strings"
)

func MultipleValidIntegers(paramname string, paramvalue interface{}) {
	if reflect.TypeOf(paramvalue).Kind() == reflect.String {
		paramvalue := paramvalue.(string)
		values := strings.SplitN(paramvalue, "<", -1)
		for i := 0; i < len(values); i++ {
			_, err := strconv.Atoi(values[i])
			if err != nil {
				log.Fatal(paramname + " Destination values in the string must be integers")
			}
		}
	} else if reflect.TypeOf(paramvalue).Kind() != reflect.Int && reflect.TypeOf(paramvalue).Kind() != reflect.String {
		log.Fatal(paramname + " must be either string or integer")
	}
}
