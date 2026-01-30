package card

func renderElements(el []Element) []Renderer {
	ret := make([]Renderer, len(el))
	for i, v := range el {
		ret[i] = v.Render()
	}
	return ret
}
