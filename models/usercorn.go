package models

import (
	"github.com/lunixbochs/ghostrace/ghost/memio"

	"github.com/felberj/binemu/models/cpu"
	"github.com/felberj/ramfs"
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
	Task
	Config() *Config
	Run() error

	Brk(addr uint64) (uint64, error)
	Mem() memio.MemIO
	MapStack(base uint64, size uint64, guard bool) error
	StrucAt(addr uint64) *StrucStream

	Exe() string
	Loader() Loader
	Base() uint64
	Entry() uint64
	BinEntry() uint64
	SetEntry(entry uint64)
	SetExit(exit uint64)
	Kernel(i int) interface{}

	HookMapAdd(mapCb MapCb, unmapCb UnmapCb, protCb ProtCb) *MapHook
	HookMapDel(cb *MapHook)

	AddKernel(kernel interface{}, first bool)
	Syscall(num int, name string, getArgs SysGetArgs) (uint64, error)

	Exit(err error)

	Fs() *ramfs.Filesystem
}
