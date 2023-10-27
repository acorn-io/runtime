//go:build !image

package appdefinition

// If the binary is being built on its own and not part of an image, then don't worry about this binary.
var acornSleepBinary []byte
