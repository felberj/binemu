package vlinux

import (
	"log"

	co "github.com/lunixbochs/usercorn/kernel/common"
	"github.com/lunixbochs/usercorn/kernel/linux/unpack"
	"github.com/lunixbochs/usercorn/models/cpu"
	"github.com/lunixbochs/usercorn/native/enum"
)

const (
	// PageSize that the memory must be aligned
	PageSize = 0x1000
)

// Brk syscall
func (k *VirtualLinuxKernel) Brk(addr uint64) uint64 {
	ret, _ := k.U.Brk(addr)
	return ret
}

// Mmap syscall
func (k *VirtualLinuxKernel) Mmap(addrHint, size uint64, prot enum.MmapProt, flags enum.MmapFlag, fd co.Fd, off co.Off) uint64 {
	// TODO: how do we request an enum lookup from the current kernel?
	MapFixed := enum.MmapFlag(0x10) // on OS X and Linux anyway
	// TODO: MAP_FIXED means abort if we can't get the address
	var (
		data     []byte
		fileDesc *cpu.FileDesc
	)
	// if there's a file descriptor, map (copy for now) the file here before messing with guest memory
	if fd > 0 {
		size := size
		fd, ok := k.Fds[fd]
		if !ok {
			log.Printf("Invalid mmap of fd %d", fd)
			return MinusOne
		}
		stat, err := fd.Stat()
		if err != nil {
			return MinusOne
		}
		fileDesc = &cpu.FileDesc{Name: stat.Name(), Off: uint64(off), Len: size}
		defer fd.Close()
		if size+uint64(off) > uint64(stat.Size()) {
			size = uint64(stat.Size()) - uint64(off)
		}
		dup, err := k.Fs.Open(stat.Name())
		if err != nil {
			return MinusOne
		}
		defer dup.Close()
		if o, err := dup.Seek(int64(off), 0); int64(off) != o || err != nil {
			return MinusOne
		}
		data = make([]byte, size)
		dup.Read(data)
	}
	fixed := flags&MapFixed != 0
	if addrHint == 0 && !fixed {
		// don't automap memory within 8MB of the current program break
		brk, _ := k.U.Brk(0)
		addrHint = brk + 0x800000
	}
	if size%PageSize != 0 {
		size += PageSize - (size % PageSize)
	}
	addr, err := k.U.Mmap(addrHint, size, int(prot), fixed, "mmap", fileDesc)
	if err != nil {
		return MinusOne
	}
	if fd > 0 && data != nil {
		err := k.U.MemWrite(addr, data)
		if err != nil {
			return MinusOne
		}
	}
	return addr
}

// Mprotect syscall
func (k *VirtualLinuxKernel) Mprotect(addr, size uint64, prot uint64) uint64 {
	if prot == 0 {
		// FIXME: Issue #137
		prot = cpu.PROT_ALL
	}
	p := unpack.MmapProt(prot)
	if err := k.U.MemProt(addr, size, int(p)); err != nil {
		return MinusOne
	}
	return 0
}
