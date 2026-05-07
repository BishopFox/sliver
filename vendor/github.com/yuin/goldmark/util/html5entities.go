package util

import (
	"sync"
)

//go:generate go run ../_tools emb-structs -i ../_tools/html5entities.json -o ./html5entities.gen.go

var _html5entitiesOnce sync.Once
var _html5entitiesMap map[string]*HTML5Entity

func buildHTML5Entities() {
	_html5entitiesOnce.Do(func() {
		entities := make([]HTML5Entity, _html5entitiesLength)
		_html5entitiesMap = make(map[string]*HTML5Entity, _html5entitiesLength)

		cName := 0
		cCharacters := 0
		for i := 0; i < _html5entitiesLength; i++ {
			tName := cName + int(_html5entitiesNameIndex[i])
			tCharacters := cCharacters + int(_html5entitiesCharactersIndex[i])

			name := _html5entitiesName[cName:tName]
			e := &entities[i]
			e.Name = name
			e.Characters = _html5entitiesCharacters[cCharacters:tCharacters]
			_html5entitiesMap[name] = e

			cName = tName
			cCharacters = tCharacters
		}
	})
}

// HTML5Entity struct represents HTML5 entitites.
type HTML5Entity struct {
	Name       string
	Characters []byte
}

// LookUpHTML5EntityByName returns (an HTML5Entity, true) if an entity named
// given name is found, otherwise (nil, false).
func LookUpHTML5EntityByName(name string) (*HTML5Entity, bool) {
	buildHTML5Entities()
	v, ok := _html5entitiesMap[name]
	return v, ok
}
