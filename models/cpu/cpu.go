package cpu

import (
	"io"

	uc "github.com/felberj/binemu/unicorn"
)

// This interface abstracts the minimum functionality Usercorn requires in a CPU emulator.
type Cpu interface {
	// memory mapping
	MemMap(addr, size uint64, prot int) error
	MemProt(addr, size uint64, prot int) error
	MemUnmap(addr, size uint64) error
	MemRegions() ([]*uc.MemRegion, error)

	// memory IO
	Mem() io.ReadWriteSeeker

	// register IO
	RegRead(reg int) (uint64, error)
	RegWrite(reg int, val uint64) error

	// execution
	Start(begin, until uint64) error
	Stop() error

	// hooks
	HookAdd(htype int, cb interface{}, begin, end uint64, extra ...int) (Hook, error)
	HookDel(hook Hook) error

	// save/restore entire CPU state
	ContextSave(reuse interface{}) (interface{}, error)
	ContextRestore(ctx interface{}) error

	// cleanup
	Close() error

	// leaky abstraction
	Backend() interface{}
}
