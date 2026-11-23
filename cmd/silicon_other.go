//go:build !amd64

package cmd

import "runtime"

func GetSiliconVendor() string {
	return "NON-X86_64 (" + runtime.GOARCH + " - ASM BYPASSED)"
}
