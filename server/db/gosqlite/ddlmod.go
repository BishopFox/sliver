package gosqlite

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type ddl struct {
	head   string
	fields []string
}

func parseDDL(sql string) (*ddl, error) {
	reg := regexp.MustCompile("(?i)(CREATE TABLE [\"`]?[\\w\\d]+[\"`]?)(?: \\((.*)\\))?")
	sections := reg.FindStringSubmatch(sql)

	if sections == nil {
		return nil, errors.New("invalid DDL")
	}

	ddlHead := sections[1]
	ddlBody := sections[2]
	ddlBodyRunes := []rune(ddlBody)
	fields := []string{}

	bracketLevel := 0
	var quote rune = 0
	buf := ""

	for i := 0; i < len(ddlBodyRunes); i++ {
		c := ddlBodyRunes[i]
		var next rune = 0
		if i+1 < len(ddlBodyRunes) {
			next = ddlBodyRunes[i+1]
		}

		if c == '\'' || c == '"' || c == '`' {
			if c == next {
				// Skip escaped quote
				buf += string(c)
				i++
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
					fields = append(fields, strings.TrimSpace(buf))
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
		fields = append(fields, strings.TrimSpace(buf))
	}

	return &ddl{head: ddlHead, fields: fields}, nil
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

// func (d *ddl) hasConstraint(name string) bool {
// 	reg := regexp.MustCompile("^CONSTRAINT [\"`]?" + regexp.QuoteMeta(name) + "[\"` ]")

// 	for _, f := range d.fields {
// 		if reg.MatchString(f) {
// 			return true
// 		}
// 	}
// 	return false
// }

func (d *ddl) getColumns() []string {
	res := []string{}

	for _, f := range d.fields {
		fUpper := strings.ToUpper(f)
		if strings.HasPrefix(fUpper, "PRIMARY KEY") ||
			strings.HasPrefix(fUpper, "CHECK") ||
			strings.HasPrefix(fUpper, "CONSTRAINT") {
			continue
		}

		reg := regexp.MustCompile("^[\"`]?([\\w\\d]+)[\"`]?")
		match := reg.FindStringSubmatch(f)

		if match != nil {
			res = append(res, "`"+match[1]+"`")
		}
	}
	return res
}
