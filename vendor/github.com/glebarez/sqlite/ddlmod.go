package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm/migrator"
)

var (
	sqliteSeparator    = "`|\"|'|\t"
	indexRegexp        = regexp.MustCompile(fmt.Sprintf(`(?is)CREATE(?: UNIQUE)? INDEX [%v]?[\w\d-]+[%v]? ON (.*)$`, sqliteSeparator, sqliteSeparator))
	tableRegexp        = regexp.MustCompile(fmt.Sprintf(`(?is)(CREATE TABLE [%v]?[\w\d-]+[%v]?)(?:\s*\((.*)\))?`, sqliteSeparator, sqliteSeparator))
	separatorRegexp    = regexp.MustCompile(fmt.Sprintf("[%v]", sqliteSeparator))
	columnsRegexp      = regexp.MustCompile(fmt.Sprintf(`[(,][%v]?(\w+)[%v]?`, sqliteSeparator, sqliteSeparator))
	columnRegexp       = regexp.MustCompile(fmt.Sprintf(`^[%v]?([\w\d]+)[%v]?\s+([\w\(\)\d]+)(.*)$`, sqliteSeparator, sqliteSeparator))
	defaultValueRegexp = regexp.MustCompile(`(?i) DEFAULT \(?(.+)?\)?( |COLLATE|GENERATED|$)`)
	regRealDataType    = regexp.MustCompile(`[^\d](\d+)[^\d]?`)
)

func getAllColumns(s string) []string {
	allMatches := columnsRegexp.FindAllStringSubmatch(s, -1)
	columns := make([]string, 0, len(allMatches))
	for _, matches := range allMatches {
		if len(matches) > 1 {
			columns = append(columns, matches[1])
		}
	}
	return columns
}

type ddl struct {
	head    string
	fields  []string
	columns []migrator.ColumnType
}

func parseDDL(strs ...string) (*ddl, error) {
	var result ddl
	for _, str := range strs {
		if sections := tableRegexp.FindStringSubmatch(str); len(sections) > 0 {
			var (
				ddlBody      = sections[2]
				ddlBodyRunes = []rune(ddlBody)
				bracketLevel int
				quote        rune
				buf          string
			)
			ddlBodyRunesLen := len(ddlBodyRunes)

			result.head = sections[1]

			for idx := 0; idx < ddlBodyRunesLen; idx++ {
				var (
					next rune = 0
					c         = ddlBodyRunes[idx]
				)
				if idx+1 < ddlBodyRunesLen {
					next = ddlBodyRunes[idx+1]
				}

				if sc := string(c); separatorRegexp.MatchString(sc) {
					if c == next {
						buf += sc // Skip escaped quote
						idx++
					} else if quote > 0 {
						quote = 0
					} else {
						quote = c
					}
				} else if quote == 0 {
					if c == '(' {
						bracketLevel++
					} else if c == ')' {
						bracketLevel--
					} else if bracketLevel == 0 {
						if c == ',' {
							result.fields = append(result.fields, strings.TrimSpace(buf))
							buf = ""
							continue
						}
					}
				}

				if bracketLevel < 0 {
					return nil, errors.New("invalid DDL, unbalanced brackets")
				}

				buf += string(c)
			}

			if bracketLevel != 0 {
				return nil, errors.New("invalid DDL, unbalanced brackets")
			}

			if buf != "" {
				result.fields = append(result.fields, strings.TrimSpace(buf))
			}

			for _, f := range result.fields {
				fUpper := strings.ToUpper(f)
				if strings.HasPrefix(fUpper, "CHECK") ||
					strings.HasPrefix(fUpper, "CONSTRAINT") {
					continue
				}

				if strings.HasPrefix(fUpper, "PRIMARY KEY") {
					for _, name := range getAllColumns(f) {
						for idx, column := range result.columns {
							if column.NameValue.String == name {
								column.PrimaryKeyValue = sql.NullBool{Bool: true, Valid: true}
								result.columns[idx] = column
								break
							}
						}
					}
				} else if matches := columnRegexp.FindStringSubmatch(f); len(matches) > 0 {
					columnType := migrator.ColumnType{
						NameValue:         sql.NullString{String: matches[1], Valid: true},
						DataTypeValue:     sql.NullString{String: matches[2], Valid: true},
						ColumnTypeValue:   sql.NullString{String: matches[2], Valid: true},
						PrimaryKeyValue:   sql.NullBool{Valid: true},
						UniqueValue:       sql.NullBool{Valid: true},
						NullableValue:     sql.NullBool{Valid: true},
						DefaultValueValue: sql.NullString{Valid: false},
					}

					matchUpper := strings.ToUpper(matches[3])
					if strings.Contains(matchUpper, " NOT NULL") {
						columnType.NullableValue = sql.NullBool{Bool: false, Valid: true}
					} else if strings.Contains(matchUpper, " NULL") {
						columnType.NullableValue = sql.NullBool{Bool: true, Valid: true}
					}
					if strings.Contains(matchUpper, " UNIQUE") {
						columnType.UniqueValue = sql.NullBool{Bool: true, Valid: true}
					}
					if strings.Contains(matchUpper, " PRIMARY") {
						columnType.PrimaryKeyValue = sql.NullBool{Bool: true, Valid: true}
					}
					if defaultMatches := defaultValueRegexp.FindStringSubmatch(matches[3]); len(defaultMatches) > 1 {
						if strings.ToLower(defaultMatches[1]) != "null" {
							columnType.DefaultValueValue = sql.NullString{String: strings.Trim(defaultMatches[1], `"`), Valid: true}
						}
					}

					// data type length
					matches := regRealDataType.FindAllStringSubmatch(columnType.DataTypeValue.String, -1)
					if len(matches) == 1 && len(matches[0]) == 2 {
						size, _ := strconv.Atoi(matches[0][1])
						columnType.LengthValue = sql.NullInt64{Valid: true, Int64: int64(size)}
						columnType.DataTypeValue.String = strings.TrimSuffix(columnType.DataTypeValue.String, matches[0][0])
					}

					result.columns = append(result.columns, columnType)
				}
			}
		} else if matches := indexRegexp.FindStringSubmatch(str); len(matches) > 0 {
			for _, column := range getAllColumns(matches[1]) {
				for idx, c := range result.columns {
					if c.NameValue.String == column {
						c.UniqueValue = sql.NullBool{Bool: strings.ToUpper(strings.Fields(str)[1]) == "UNIQUE", Valid: true}
						result.columns[idx] = c
					}
				}
			}
		} else {
			return nil, errors.New("invalid DDL")
		}
	}

	return &result, nil
}

func (d *ddl) compile() string {
	if len(d.fields) == 0 {
		return d.head
	}

	return fmt.Sprintf("%s (%s)", d.head, strings.Join(d.fields, ","))
}

func (d *ddl) addConstraint(name string, sql string) {
	reg := regexp.MustCompile("^CONSTRAINT [\"`]?" + regexp.QuoteMeta(name) + "[\"` ]")

	for i := 0; i < len(d.fields); i++ {
		if reg.MatchString(d.fields[i]) {
			d.fields[i] = sql
			return
		}
	}

	d.fields = append(d.fields, sql)
}

func (d *ddl) removeConstraint(name string) bool {
	reg := regexp.MustCompile("^CONSTRAINT [\"`]?" + regexp.QuoteMeta(name) + "[\"` ]")

	for i := 0; i < len(d.fields); i++ {
		if reg.MatchString(d.fields[i]) {
			d.fields = append(d.fields[:i], d.fields[i+1:]...)
			return true
		}
	}
	return false
}

func (d *ddl) hasConstraint(name string) bool {
	reg := regexp.MustCompile("^CONSTRAINT [\"`]?" + regexp.QuoteMeta(name) + "[\"` ]")

	for _, f := range d.fields {
		if reg.MatchString(f) {
			return true
		}
	}
	return false
}

func (d *ddl) getColumns() []string {
	res := []string{}

	for _, f := range d.fields {
		fUpper := strings.ToUpper(f)
		if strings.HasPrefix(fUpper, "PRIMARY KEY") ||
			strings.HasPrefix(fUpper, "CHECK") ||
			strings.HasPrefix(fUpper, "CONSTRAINT") ||
			strings.Contains(fUpper, "GENERATED ALWAYS AS") {
			continue
		}

		reg := regexp.MustCompile("^[\"`']?([\\w\\d]+)[\"`']?")
		match := reg.FindStringSubmatch(f)

		if match != nil {
			res = append(res, "`"+match[1]+"`")
		}
	}
	return res
}
