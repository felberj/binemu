package linux

import (
	co "github.com/lunixbochs/usercorn/kernel/common"
	"github.com/lunixbochs/usercorn/native/enum"
)

func (k *LinuxKernel) Mmap2(addrHint, size uint64, prot enum.MmapProt, flags enum.MmapFlag, fd co.Fd, off co.Off) uint64 {
	return k.Mmap(addrHint, size, prot, flags, fd, off*0x1000)
}
