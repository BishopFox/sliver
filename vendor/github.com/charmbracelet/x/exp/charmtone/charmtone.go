// Package charmtone contains an API for the CharmTone color palette.
package charmtone

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"
)

var _ color.Color = Key(0)

// Key is a type for color keys.
type Key int

// Available colors.
const (
	Cumin Key = iota
	Tang
	Yam
	Paprika
	Bengal
	Uni
	Sriracha
	Coral
	Salmon
	Chili
	Cherry
	Tuna
	Macaron
	Pony
	Cheeky
	Flamingo
	Dolly
	Blush
	Urchin
	Mochi
	Lilac
	Prince
	Violet
	Mauve
	Grape
	Plum
	Orchid
	Jelly
	Charple
	Hazy
	Ox
	Sapphire
	Guppy
	Oceania
	Thunder
	Anchovy
	Damson
	Malibu
	Sardine
	Zinc
	Turtle
	Lichen
	Guac
	Julep
	Bok
	Mustard
	Citron
	Zest
	Pepper
	BBQ
	Charcoal
	Iron
	Oyster
	Squid
	Smoke
	Ash
	Salt
	Butter

	// Diffs: additions. The brightest color in this set is Julep, defined
	// above.
	Pickle
	Gator
	Spinach

	// Diffs: deletions. The brightest color in this set is Cherry, defined
	// above.
	Pom
	Steak
	Toast

	// Provisional.
	NeueGuac
	NeueZinc
)

// RGBA returns the red, green, blue, and alpha values of the color. It
// satisfies the color.Color interface.
func (k Key) RGBA() (r, g, b, a uint32) {
	c, ok := colors[k]
	if !ok {
		panic("invalid color key " + strconv.Itoa(int(k)))
	}
	return c.RGBA()
}

var names = map[Key]string{
	Cumin:    "Cumin",
	Tang:     "Tang",
	Yam:      "Yam",
	Paprika:  "Paprika",
	Bengal:   "Bengal",
	Uni:      "Uni",
	Sriracha: "Sriracha",
	Coral:    "Coral",
	Salmon:   "Salmon",
	Chili:    "Chili",
	Cherry:   "Cherry",
	Tuna:     "Tuna",
	Macaron:  "Macaron",
	Pony:     "Pony",
	Cheeky:   "Cheeky",
	Flamingo: "Flamingo",
	Dolly:    "Dolly",
	Blush:    "Blush",
	Urchin:   "Urchin",
	Mochi:    "Crystal",
	Lilac:    "Lilac",
	Prince:   "Prince",
	Violet:   "Violet",
	Mauve:    "Mauve",
	Grape:    "Grape",
	Plum:     "Plum",
	Orchid:   "Orchid",
	Jelly:    "Jelly",
	Charple:  "Charple",
	Hazy:     "Hazy",
	Ox:       "Ox",
	Sapphire: "Sapphire",
	Guppy:    "Guppy",
	Oceania:  "Oceania",
	Thunder:  "Thunder",
	Anchovy:  "Anchovy",
	Damson:   "Damson",
	Malibu:   "Malibu",
	Sardine:  "Sardine",
	Zinc:     "Zinc",
	Turtle:   "Turtle",
	Lichen:   "Lichen",
	Guac:     "Guac",
	Julep:    "Julep",
	Bok:      "Bok",
	Mustard:  "Mustard",
	Citron:   "Citron",
	Zest:     "Zest",
	Pepper:   "Pepper",
	BBQ:      "BBQ",
	Charcoal: "Charcoal",
	Iron:     "Iron",
	Oyster:   "Oyster",
	Squid:    "Squid",
	Smoke:    "Smoke",
	Salt:     "Salt",
	Ash:      "Ash",
	Butter:   "Butter",

	// Diffs: additions.
	Pickle:  "Pickle",
	Gator:   "Gator",
	Spinach: "Spinach",

	// Diffs: deletions.
	Pom:   "Pom",
	Steak: "Steak",
	Toast: "Toast",

	// Provisional.
	NeueGuac: "Neue Guac",
	NeueZinc: "Neue Zinc",
}

// String returns the official CharmTone name of the color. It satisfies the
// fmt.Stringer interface.
func (k Key) String() string {
	name, ok := names[k]
	if !ok {
		return ""
	}
	return name
}

var colors = map[Key]color.RGBA{
	Cumin:    {R: 0xBF, G: 0x97, B: 0x6F, A: 0xFF}, // "#BF976F"
	Tang:     {R: 0xFF, G: 0x98, B: 0x5A, A: 0xFF}, // "#FF985A"
	Yam:      {R: 0xFF, G: 0xB5, B: 0x87, A: 0xFF}, // "#FFB587"
	Paprika:  {R: 0xD3, G: 0x6C, B: 0x64, A: 0xFF}, // "#D36C64"
	Bengal:   {R: 0xFF, G: 0x6E, B: 0x63, A: 0xFF}, // "#FF6E63"
	Uni:      {R: 0xFF, G: 0x93, B: 0x7D, A: 0xFF}, // "#FF937D"
	Sriracha: {R: 0xEB, G: 0x42, B: 0x68, A: 0xFF}, // "#EB4268"
	Coral:    {R: 0xFF, G: 0x57, B: 0x7D, A: 0xFF}, // "#FF577D"
	Salmon:   {R: 0xFF, G: 0x7F, B: 0x90, A: 0xFF}, // "#FF7F90"
	Chili:    {R: 0xE2, G: 0x30, B: 0x80, A: 0xFF}, // "#E23080"
	Cherry:   {R: 0xFF, G: 0x38, B: 0x8B, A: 0xFF}, // "#FF388B"
	Tuna:     {R: 0xFF, G: 0x6D, B: 0xAA, A: 0xFF}, // "#FF6DAA"
	Macaron:  {R: 0xE9, G: 0x40, B: 0xB0, A: 0xFF}, // "#E940B0"
	Pony:     {R: 0xFF, G: 0x4F, B: 0xBF, A: 0xFF}, // "#FF4FBF"
	Cheeky:   {R: 0xFF, G: 0x79, B: 0xD0, A: 0xFF}, // "#FF79D0"
	Flamingo: {R: 0xF9, G: 0x47, B: 0xE3, A: 0xFF}, // "#F947E3"
	Dolly:    {R: 0xFF, G: 0x60, B: 0xFF, A: 0xFF}, // "#FF60FF"
	Blush:    {R: 0xFF, G: 0x84, B: 0xFF, A: 0xFF}, // "#FF84FF"
	Urchin:   {R: 0xC3, G: 0x37, B: 0xE0, A: 0xFF}, // "#C337E0"
	Mochi:    {R: 0xEB, G: 0x5D, B: 0xFF, A: 0xFF}, // "#EB5DFF"
	Lilac:    {R: 0xF3, G: 0x79, B: 0xFF, A: 0xFF}, // "#F379FF"
	Prince:   {R: 0x9C, G: 0x35, B: 0xE1, A: 0xFF}, // "#9C35E1"
	Violet:   {R: 0xC2, G: 0x59, B: 0xFF, A: 0xFF}, // "#C259FF"
	Mauve:    {R: 0xD4, G: 0x6E, B: 0xFF, A: 0xFF}, // "#D46EFF"
	Grape:    {R: 0x71, G: 0x34, B: 0xDD, A: 0xFF}, // "#7134DD"
	Plum:     {R: 0x99, G: 0x53, B: 0xFF, A: 0xFF}, // "#9953FF"
	Orchid:   {R: 0xAD, G: 0x6E, B: 0xFF, A: 0xFF}, // "#AD6EFF"
	Jelly:    {R: 0x4A, G: 0x30, B: 0xD9, A: 0xFF}, // "#4A30D9"
	Charple:  {R: 0x6B, G: 0x50, B: 0xFF, A: 0xFF}, // "#6B50FF"
	Hazy:     {R: 0x8B, G: 0x75, B: 0xFF, A: 0xFF}, // "#8B75FF"
	Ox:       {R: 0x33, G: 0x31, B: 0xB2, A: 0xFF}, // "#3331B2"
	Sapphire: {R: 0x49, G: 0x49, B: 0xFF, A: 0xFF}, // "#4949FF"
	Guppy:    {R: 0x72, G: 0x72, B: 0xFF, A: 0xFF}, // "#7272FF"
	Oceania:  {R: 0x2B, G: 0x55, B: 0xB3, A: 0xFF}, // "#2B55B3"
	Thunder:  {R: 0x47, G: 0x76, B: 0xFF, A: 0xFF}, // "#4776FF"
	Anchovy:  {R: 0x71, G: 0x9A, B: 0xFC, A: 0xFF}, // "#719AFC"
	Damson:   {R: 0x00, G: 0x7A, B: 0xB8, A: 0xFF}, // "#007AB8"
	Malibu:   {R: 0x00, G: 0xA4, B: 0xFF, A: 0xFF}, // "#00A4FF"
	Sardine:  {R: 0x4F, G: 0xBE, B: 0xFE, A: 0xFF}, // "#4FBEFE"
	Zinc:     {R: 0x10, G: 0xB1, B: 0xAE, A: 0xFF}, // "#10B1AE"
	Turtle:   {R: 0x0A, G: 0xDC, B: 0xD9, A: 0xFF}, // "#0ADCD9"
	Lichen:   {R: 0x5C, G: 0xDF, B: 0xEA, A: 0xFF}, // "#5CDFEA"
	Guac:     {R: 0x12, G: 0xC7, B: 0x8F, A: 0xFF}, // "#12C78F"
	Julep:    {R: 0x00, G: 0xFF, B: 0xB2, A: 0xFF}, // "#00FFB2"
	Bok:      {R: 0x68, G: 0xFF, B: 0xD6, A: 0xFF}, // "#68FFD6"
	Mustard:  {R: 0xF5, G: 0xEF, B: 0x34, A: 0xFF}, // "#F5EF34"
	Citron:   {R: 0xE8, G: 0xFF, B: 0x27, A: 0xFF}, // "#E8FF27"
	Zest:     {R: 0xE8, G: 0xFE, B: 0x96, A: 0xFF}, // "#E8FE96"
	Pepper:   {R: 0x20, G: 0x1F, B: 0x26, A: 0xFF}, // "#201F26"
	BBQ:      {R: 0x2d, G: 0x2c, B: 0x35, A: 0xFF}, // "#2d2c35"
	Charcoal: {R: 0x3A, G: 0x39, B: 0x43, A: 0xFF}, // "#3A3943"
	Iron:     {R: 0x4D, G: 0x4C, B: 0x57, A: 0xFF}, // "#4D4C57"
	Oyster:   {R: 0x60, G: 0x5F, B: 0x6B, A: 0xFF}, // "#605F6B"
	Squid:    {R: 0x85, G: 0x83, B: 0x92, A: 0xFF}, // "#858392"
	Smoke:    {R: 0xBF, G: 0xBC, B: 0xC8, A: 0xFF}, // "#BFBCC8"
	Ash:      {R: 0xDF, G: 0xDB, B: 0xDD, A: 0xFF}, // "#DFDBDD"
	Salt:     {R: 0xF1, G: 0xEF, B: 0xEF, A: 0xFF}, // "#F1EFEF"
	Butter:   {R: 0xFF, G: 0xFA, B: 0xF1, A: 0xFF}, // "#FFFAF1"

	// Diffs: additions.
	Pickle:  {R: 0x00, G: 0xA4, B: 0x75, A: 0xFF}, // "#00A475"
	Gator:   {R: 0x18, G: 0x46, B: 0x3D, A: 0xFF}, // "#18463D"
	Spinach: {R: 0x1C, G: 0x36, B: 0x34, A: 0xFF}, // "#1C3634"

	// Diffs: deletions.
	Pom:   {R: 0xAB, G: 0x24, B: 0x54, A: 0xFF}, // "#AB2454"
	Steak: {R: 0x58, G: 0x22, B: 0x38, A: 0xFF}, // "#582238"
	Toast: {R: 0x41, G: 0x21, B: 0x30, A: 0xFF}, // "#412130"

	// Provisional.
	NeueGuac: {R: 0x00, G: 0xb8, B: 0x75, A: 0xFF}, // "#00b875"
	NeueZinc: {R: 0x0e, G: 0x99, B: 0x96, A: 0xFF}, // "#0e9996"
}

// Hex returns the hex value of the color.
func (k Key) Hex() string {
	c, ok := colors[k]
	if !ok {
		panic("invalid color key " + strconv.Itoa(int(k)))
	}
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// Keys returns a slice of all CharmTone color keys.
func Keys() []Key {
	return []Key{
		Cumin,
		Tang,
		Yam,
		Paprika,
		Bengal,
		Uni,
		Sriracha,
		Coral,
		Salmon,
		Chili,
		Cherry,
		Tuna,
		Macaron,
		Pony,
		Cheeky,
		Flamingo,
		Dolly,
		Blush,
		Urchin,
		Mochi,
		Lilac,
		Prince,
		Violet,
		Mauve,
		Grape,
		Plum,
		Orchid,
		Jelly,
		Charple,
		Hazy,
		Ox,
		Sapphire,
		Guppy,
		Oceania,
		Thunder,
		Anchovy,
		Damson,
		Malibu,
		Sardine,
		Zinc,
		Turtle,
		Lichen,
		Guac,
		Julep,
		Bok,
		Mustard,
		Citron,
		Zest,
		Pepper,
		BBQ,
		Charcoal,
		Iron,
		Oyster,
		Squid,
		Smoke,
		Ash,
		Salt,
		Butter,

		// XXX: additions and deletions are not included, yet.
	}
}

// IsPrimary indicates which colors are part of the core palette.
func (k Key) IsPrimary() bool {
	return slices.Contains([]Key{
		Charple,
		Dolly,
		Julep,
		Zest,
		Butter,
	}, k)
}

// IsSecondary indicates which colors are part of the secondary palette.
func (k Key) IsSecondary() bool {
	return slices.Contains([]Key{
		Hazy,
		Blush,
		Bok,
	}, k)
}

// IsTertiary indicates which colors are part of the tertiary palette.
func (k Key) IsTertiary() bool {
	return slices.Contains([]Key{
		Turtle,
		Malibu,
		Violet,
		Tuna,
		Coral,
		Uni,
	}, k)
}
