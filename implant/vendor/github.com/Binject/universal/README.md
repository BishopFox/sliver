# The Universal Loader

This loader provides a unified Go interface for loading shared libraries **from memory** on Windows, OSX, and Linux.

Also included is a cross-platform `Call()` implementation that lets you call into exported symbols from those libraries without stress.

### Basic Usage

*libraryPath* set to `lib.so` for Linux, `lib.dyld` for OSX, or `lib.DLL` for Windows, then:

```
	image, err = ioutil.ReadFile(libraryPath)
	...

	loader, err := universal.NewLoader()
	...

	library, err := loader.LoadLibrary("main", &image)
	...

	val, err := library.Call("Runme", 7)
	...
```

Complete working examples of usage can be found in the examples/ folder in this repo.


### Features and Limitations
- OSX backend uses the system loader, so multiple interdependent libraries can be loaded.
- OSX backend **automatically adds the underscore prefix for you**, so you can refer to symbols the same way on Linux, Windows, and OSX.
- Linux and Windows backends do not use the system loader, so interdependent libraries cannot be loaded.
- Linux backend does not use memfd!  
***I believe this is the first Golang Linux loader that does not use memfd!***

### Supported Architectures
- OSX arm64 M1 Apple Chip (tested)  
***This is the first loader I've seen, Golang or not, that works on the M1!***
- OSX amd64     (tested)
- Windows amd64 (tested)
- Windows 386   (untested, should work)
- Linux amd64   (tested)
- Linux 386     (untested, should work) 
- Other POSIX systems on 386, amd64, or arm64 (untested, should work)
  
**If you try this on any untested platforms, whether it works or not, let me know!**
