package driver

import "database/sql/driver"

func namedValues(args []driver.Value) []driver.NamedValue {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}
	return named
}
