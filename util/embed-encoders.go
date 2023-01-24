package util

import (
	"embed"
)

//
// This embeds the source code for the encoders into the server
// binary so that we can render it along with the implant source
//

//go:embed encoders/*
var EncodersSrc embed.FS
