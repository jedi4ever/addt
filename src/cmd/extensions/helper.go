package extensions

import (
	"github.com/jedi4ever/addt/extensions"
)

// GetEntrypoint returns the entrypoint command for a given extension name
// If extension not found, returns the extension name itself as fallback
func GetEntrypoint(extName string) string {
	exts, err := extensions.GetExtensions()
	if err != nil {
		return extName
	}

	for _, ext := range exts {
		if ext.Name == extName {
			if cmd := ext.Entrypoint.Command(); cmd != "" {
				return cmd
			}
			return extName
		}
	}

	return extName
}

// Exists checks if an extension with the given name exists
func Exists(name string) bool {
	exts, err := extensions.GetExtensions()
	if err != nil {
		return false
	}
	for _, ext := range exts {
		if ext.Name == name {
			return true
		}
	}
	return false
}
