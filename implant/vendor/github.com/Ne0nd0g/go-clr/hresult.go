package clr

// https://docs.microsoft.com/en-us/dotnet/framework/interop/how-to-map-hresults-and-exceptions
// https://docs.microsoft.com/en-us/windows/win32/seccrypto/common-hresult-values

const (
	S_OK    = 0x00
	S_FALSE = 0x01
	// COR_E_TARGETINVOCATION is TargetInvocationException
	// https://docs.microsoft.com/en-us/dotnet/api/system.reflection.targetinvocationexception?view=net-5.0
	COR_E_TARGETINVOCATION uint32 = 0x80131604
	// COR_E_SAFEARRAYRANKMISMATCH is SafeArrayRankMismatchException
	COR_E_SAFEARRAYRANKMISMATCH uint32 = 0x80131538
	// COR_E_BADIMAGEFORMAT is BadImageFormatException
	COR_E_BADIMAGEFORMAT uint32 = 0x8007000b
	// DISP_E_BADPARAMCOUNT is invalid number of parameters
	DISP_E_BADPARAMCOUNT uint32 = 0x8002000e
	// E_POINTER Pointer that is not valid
	E_POINTER uint32 = 0x80004003
	// E_NOINTERFACE No such interface supported
	E_NOINTERFACE uint32 = 0x80004002
)
