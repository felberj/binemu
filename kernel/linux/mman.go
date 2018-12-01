package linux

import (
	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/native/enum"
)

func (k *LinuxKernel) Mmap2(addrHint, size uint64, prot enum.MmapProt, flags enum.MmapFlag, fd co.Fd, off co.Off) uint64 {
	return k.Mmap(addrHint, size, prot, flags, fd, off*0x1000)
}
