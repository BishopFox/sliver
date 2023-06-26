//go:build !(darwin || linux || freebsd)

package platform

func remapCodeSegmentAMD64(code []byte, size int) ([]byte, error) {
	b, err := mmapCodeSegmentAMD64(size)
	if err != nil {
		return nil, err
	}
	copy(b, code)
	mustMunmapCodeSegment(code)
	return b, nil
}

func remapCodeSegmentARM64(code []byte, size int) ([]byte, error) {
	b, err := mmapCodeSegmentARM64(size)
	if err != nil {
		return nil, err
	}
	copy(b, code)
	mustMunmapCodeSegment(code)
	return b, nil
}
