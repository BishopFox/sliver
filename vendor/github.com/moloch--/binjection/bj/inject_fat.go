package bj

/* todo - this incomplete functionality should be a configurable option
		under the Macho injector, FAT is not a top-level exe type
// FatBinject - Inject a malicious macho binary into a fat file and currupt the cpu execution of the first file
func FatBinject(sourceBytes []byte, bdMachoPath string, config *BinjectConfig) error {
	//
	// Open Fat File, Read Both Macho Types
	var FatyMachos []macho.File

	fatFile, err := macho.OpenFat(sourceFile)
	if err != nil {
		return err
	}
	for _, arch := range fatFile.Arches {
		log.Printf("Arch details: %v\n", arch)
		machoFile := arch.File
		FatyMachos = append(FatyMachos, *machoFile)
	}
	log.Printf("FatyMacho Deets: %+v", FatyMachos)

	// Read Malicious Macho Type
	machoFile, err := macho.Open(bdMachoPath)
	if err != nil {
		return err
	}
	log.Printf("Malicious Macho Deets: %+v", machoFile)

	// Replace one macho, and currpt cpu
	FatyMachos[len(FatyMachos)-1] = *machoFile
	log.Printf("New FatyMacho Deets: %+v", FatyMachos)

	// Write final strut
	final, err := os.Create(destFile)
	if err != nil {
		return err
	}
	// Write fat headers
	// Write arch load headers
	// write machos
	defer final.Close()
	return nil

}
*/
