package cpu

import (
	"io"

	"github.com/pkg/errors"

	uc "github.com/felberj/binemu/cpu/unicorn"
)

// these errors are used for HOOK_MEM_ERR
const (
	MEM_WRITE_UNMAPPED = uc.MEM_WRITE_UNMAPPED
	MEM_READ_UNMAPPED  = uc.MEM_READ_UNMAPPED
	MEM_FETCH_UNMAPPED = uc.MEM_FETCH_UNMAPPED
	MEM_WRITE_PROT     = uc.MEM_WRITE_PROT
	MEM_READ_PROT      = uc.MEM_READ_PROT
	MEM_FETCH_PROT     = uc.MEM_FETCH_PROT

	MEM_PROT     = MEM_WRITE_PROT | MEM_READ_PROT | MEM_FETCH_PROT
	MEM_UNMAPPED = MEM_READ_UNMAPPED | MEM_WRITE_UNMAPPED | MEM_FETCH_UNMAPPED
)

// these constants are used for memory protections
const (
	PROT_NONE  = uc.PROT_NONE
	PROT_READ  = uc.PROT_READ
	PROT_WRITE = uc.PROT_WRITE
	PROT_EXEC  = uc.PROT_EXEC
	PROT_ALL   = uc.PROT_ALL
)

// these constants are used in a hook to specify the type of memory access
const (
	MEM_WRITE = uc.MEM_WRITE
	MEM_READ  = uc.MEM_READ
	MEM_FETCH = uc.MEM_FETCH
)

type memFile struct {
	off uint64
	u   *uc.Unicorn
}

func (m *memFile) Read(p []byte) (int, error) {
	err := m.u.MemReadInto(p, m.off)
	m.off += uint64(len(p))
	return len(p), err
}

func (m *memFile) Write(p []byte) (int, error) {
	err := m.u.MemWrite(m.off, p)
	m.off += uint64(len(p))
	return len(p), err
}

func (m *memFile) Seek(off int64, whence int) (int64, error) {
	offset := uint64(off)
	switch whence {
	default:
		return 0, errors.New("Seek: invalid whence")
	case io.SeekStart:
		// offset += s.base
		// pass
	case io.SeekCurrent:
		offset += m.off
		//	case io.SeekEnd:
		//		offset += s.limit
	}
	m.off = offset
	return int64(offset), nil
}

// Mem gives access to the memory of the process.
func (c *Cpu) Mem() io.ReadWriteSeeker {
	return &memFile{u: c.Unicorn}
}
