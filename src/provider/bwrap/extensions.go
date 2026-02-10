package bwrap

// GetExtensionEnvVars returns nil for bwrap — there are no image-based extensions.
// Extensions must be pre-installed on the host. Environment variables from the
// host are controlled via the core layer's BuildEnvironment.
func (b *BwrapProvider) GetExtensionEnvVars(imageName string) []string {
	return nil
}
