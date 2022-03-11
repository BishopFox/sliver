package d3d

type iUnknownVtbl struct {
	// every COM object starts with these three
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// _QueryInterface2 uintptr
}
