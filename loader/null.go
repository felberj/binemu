package loader

import (
	"encoding/binary"

	"github.com/felberj/binemu/models"
)

type NullLoader struct {
	LoaderBase
}

func NewNullLoader(arch, os string, byteOrder binary.ByteOrder, entry uint64) models.Loader {
	return &NullLoader{LoaderBase{
		arch:      arch,
		os:        os,
		byteOrder: byteOrder,
		entry:     entry,
	}}
}
