package unicorn

import (
	"io"

	"github.com/pkg/errors"

	uc "github.com/felberj/binemu/unicorn"
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
