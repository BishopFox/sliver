package universal

// Library - describes a loaded library
type Library struct {
	Name        string
	BaseAddress uintptr
	Exports     map[string]uint64
}

// Loader - keeps track of
type Loader struct {
	Libraries []*Library
}

// NewLoader - returns a new instance of a Loader
func NewLoader() (*Loader, error) {
	return &Loader{}, nil
}

// LoadLibrary - loads a library into this process from the given buffer
func (l *Loader) LoadLibrary(name string, image *[]byte) (*Library, error) {

	library, err := LoadLibraryImpl(name, image)
	if err != nil {
		return nil, err
	}
	library.Name = name
	l.Libraries = append(l.Libraries, library)
	return library, nil
}

// FindProc - returns the address of the given function from the given library
func (l *Loader) FindProc(libname string, funcname string) (uintptr, bool) {

	for _, lib := range l.Libraries {
		if lib.Name == libname {
			return lib.FindProc(funcname)
		}
	}
	return 0, false
}

// FindProc - returns the address of the given function in this library
func (l *Library) FindProc(funcname string) (uintptr, bool) {
	v, ok := l.Exports[funcname]
	return l.BaseAddress + uintptr(v), ok
}
