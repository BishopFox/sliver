package gen

// cross returns the cross product of the lists in x.
func cross(x [][]string) [][]string {
	r := [][]string{nil}
	for _, s := range x {
		var nxt [][]string
		for _, pre := range r {
			for _, a := range s {
				concat := make([]string, len(pre), len(pre)+1)
				copy(concat, pre)
				concat = append(concat, a)
				nxt = append(nxt, concat)
			}
		}
		r = nxt
	}
	return r
}
