//go:build amd64

package cmd

import "encoding/binary"

func rawCPUID(eaxIn, ecxIn uint32) (eax, ebx, ecx, edx uint32)

func GetSiliconVendor() string {
_, ebx, ecx, edx := rawCPUID(0, 0)
buf := make([]byte, 12)
binary.LittleEndian.PutUint32(buf[0:4], ebx)
binary.LittleEndian.PutUint32(buf[4:8], edx)
binary.LittleEndian.PutUint32(buf[8:12], ecx)
return string(buf)
}
