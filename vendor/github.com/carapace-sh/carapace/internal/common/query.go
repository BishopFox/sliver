package common

import "sort"

type Queries []string

func (q *Queries) Merge(other Queries) { // TODO pointer needed?
	for _, query := range other {
		q.Add(query)
	}
}

func (q *Queries) Add(query string) {
	unique := make(map[string]bool)
	for _, elem := range *q {
		unique[elem] = true
	}
	if _, ok := unique[query]; !ok {
		*q = append(*q, query)
		sort.Strings(*q)
	}
}

func (q Queries) Modify(s string) {
	for index, value := range q {
		q[index] = string(value) + " ||| " + s
	}
}
