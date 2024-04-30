//go:build (darwin || linux || freebsd) && !tinygo

package platform

func remapCodeSegmentAMD64(code []byte, size int) ([]byte, error) {
	return remapCodeSegment(code, size, mmapProtAMD64)
}

func remapCodeSegmentARM64(code []byte, size int) ([]byte, error) {
	return remapCodeSegment(code, size, mmapProtARM64)
}

func remapCodeSegment(code []byte, size, prot int) ([]byte, error) {
	b, err := mmapCodeSegment(size, prot)
	if err != nil {
		return nil, err
	}
	copy(b, code)
	mustMunmapCodeSegment(code)
	return b, nil
}
