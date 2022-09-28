// +build windows

package jibberjabber

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TODO: read LOCALE_NAME_MAX_LENGTH from Windows, instead of hard-coding '85'
const LOCALE_NAME_MAX_LENGTH uint32 = 85

func getWindowsLocaleFrom(sysCall string) (string, error) {
	buffer := make([]uint16, LOCALE_NAME_MAX_LENGTH)

	dll, err := windows.LoadDLL("kernel32")
	if err != nil {
		return "", errors.New("could not find kernel32 dll: " + err.Error())
	}

	proc, err := dll.FindProc(sysCall)
	if err != nil {
		return "", err
	}

	r, _, dllError := proc.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(LOCALE_NAME_MAX_LENGTH))
	if r == 0 {
		return "", errors.New(ErrLangDetectFail.Error() + ": " + dllError.Error())
	}

	return windows.UTF16ToString(buffer), nil
}

func getWindowsLocale() (string, error) {
	locale, err := getWindowsLocaleFrom("GetUserDefaultLocaleName")
	if err != nil {
		locale, err = getWindowsLocaleFrom("GetSystemDefaultLocaleName")
	}
	return locale, err
}

// DetectIETF detects and returns the IETF language tag of Windows.
func DetectIETF() (string, error) {
	return getWindowsLocale()
}
