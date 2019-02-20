package models

import (
	"encoding/binary"
	"io"

	"github.com/felberj/binemu/cpu"
	"github.com/felberj/ramfs"

	uc "github.com/felberj/binemu/cpu/unicorn"
)

type SysGetArgs func(n int) ([]uint64, error)
type SysCb func(num int, name string, args []uint64, ret uint64, desc string) bool
type SysHook struct {
	Before, After SysCb
}

type MapCb func(addr, size uint64, prot int, desc string, file *cpu.FileDesc)
type UnmapCb func(addr, size uint64)
type ProtCb func(addr, size uint64, prot int)
type MapHook struct {
	Map   MapCb
	Unmap UnmapCb
	Prot  ProtCb
}

type Usercorn interface {
	// CPU
	GetCPU() *cpu.Cpu
	Backend() interface{}
	Mem() io.ReadWriteSeeker
	RegRead(reg int) (uint64, error)
	RegWrite(reg int, val uint64) error
	MemMap(addr, size uint64, prot int) error
	MemProt(addr, size uint64, prot int) error
	MemUnmap(addr, size uint64) error
	MemRegions() ([]*uc.MemRegion, error)
	// end CPU
	//	Task
	Arch() *Arch
	ByteOrder() binary.ByteOrder
	RegDump() ([]RegVal, error)
	Bits() uint
	PackAddr(buf []byte, n uint64) ([]byte, error)
	UnpackAddr(buf []byte) uint64
	PopBytes(p []byte) error
	PushBytes(p []byte) (uint64, error)
	Pop() (uint64, error)
	Push(n uint64) (uint64, error)
	OS() string

	MemReserve(addr, size uint64, force bool) (*cpu.Page, error)
	Mmap(addr, size uint64, prot int, fixed bool, desc string, file *cpu.FileDesc) (uint64, error)
	Malloc(size uint64, desc string) (uint64, error)
	// end task

	Config() *Config
	Run() error

	Brk(addr uint64) (uint64, error)
	MapStack(base uint64, size uint64, guard bool) error
	StrucAt(addr uint64) *StrucStream

	Exe() string
	Loader() Loader
	Base() uint64
	Entry() uint64
	BinEntry() uint64
	SetEntry(entry uint64)
	SetExit(exit uint64)

	HookMapAdd(mapCb MapCb, unmapCb UnmapCb, protCb ProtCb) *MapHook
	HookMapDel(cb *MapHook)

	Syscall(num int, name string, getArgs SysGetArgs) (uint64, error)

	Exit(err error)

	Fs() *ramfs.Filesystem
}
