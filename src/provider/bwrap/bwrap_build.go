package bwrap

// BuildIfNeeded is a no-op for bwrap — there are no images to build.
// Bwrap uses the host's installed tools directly.
func (b *BwrapProvider) BuildIfNeeded(rebuild bool, rebuildBase bool) error {
	return nil
}

// DetermineImageName returns an empty string — bwrap does not use images.
func (b *BwrapProvider) DetermineImageName() string {
	return ""
}
