package loader

import (
	"encoding/binary"
)

type NullLoader struct {
	LoaderBase
}

func NewNullLoader(arch, os string, byteOrder binary.ByteOrder, entry uint64) Loader {
	return &NullLoader{LoaderBase{
		arch:      arch,
		os:        os,
		byteOrder: byteOrder,
		entry:     entry,
	}}
}
