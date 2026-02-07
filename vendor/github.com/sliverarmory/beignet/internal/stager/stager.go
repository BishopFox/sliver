package stager

// LoaderText returns the embedded darwin/arm64 loader image bytes and the entry
// function offset (relative to the start of the image).
func LoaderText() ([]byte, uint64, error) {
	text, entryOff := loaderTextDarwinArm64()
	return text, entryOff, nil
}
