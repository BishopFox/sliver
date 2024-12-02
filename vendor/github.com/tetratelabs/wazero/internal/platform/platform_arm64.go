package platform

// init verifies that the current CPU supports the required ARM64 features
func init() {
	// Ensure atomic instructions are supported.
	archRequirementsVerified = CpuFeatures.Has(CpuFeatureArm64Atomic)
}
