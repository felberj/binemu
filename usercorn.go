package usercorn

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"

	"github.com/felberj/binemu/loader"
	"github.com/felberj/binemu/models"
	"github.com/felberj/binemu/models/cpu"
	"github.com/felberj/ramfs"
	"github.com/lunixbochs/struc"
	"github.com/pkg/errors"

	co "github.com/felberj/binemu/kernel/common"
)

// #cgo LDFLAGS: -Wl,-rpath,\$ORIGIN/deps/lib:\$ORIGIN/lib
import "C"

type tramp struct {
	desc string
	fun  func() error
}

type Usercorn struct {
	*Task

	config  *models.Config
	exe     string
	loader  models.Loader
	kernels []co.Kernel

	base       uint64
	interpBase uint64
	entry      uint64
	exit       uint64
	binEntry   uint64

	StackBase uint64
	StackSize uint64
	brk       uint64

	final      sync.Once
	exitStatus error

	running     bool
	trampolines []tramp
	trampolined bool
	stackinit   bool

	restart func(models.Usercorn, error) error

	hooks []cpu.Hook
	fs    *ramfs.Filesystem
}

// NewUsercornWrapper is just a hacky woraround that usercorn has privat fields.
// TODO(felberj) remove
func NewUsercornWrapper(exe string, t *Task, fs *ramfs.Filesystem, l models.Loader, os *models.OS, c *models.Config) *Usercorn {
	u := &Usercorn{
		Task:   t,
		config: c,
		loader: l,
		exit:   0xffffffffffffffff,
		fs:     fs,
	}
	u.exe, _ = filepath.Abs(exe)

	var kernels []co.Kernel
	if os.Kernels != nil {
		kernelI := os.Kernels(u)
		for _, k := range kernelI {
			kernels = append(kernels, k.(co.Kernel))
		}
	}
	u.kernels = kernels
	return u
}

// LoadBinary is just a hacky workaround to load the binary into memory.
func (u *Usercorn) LoadBinary(f *os.File) error {
	var err error
	u.entry, u.base, u.binEntry, err = u.mapBinary(f)
	if err != nil {
		return err
	}
	// find data segment for brk
	u.brk = 0
	segments, err := u.loader.Segments()
	if err != nil {
		return err
	}
	for _, seg := range segments {
		if seg.Prot&cpu.PROT_WRITE != 0 {
			addr := u.base + seg.Addr + seg.Size
			if addr > u.brk {
				u.brk = addr
			}
		}
	}
	// TODO: have a "host page size", maybe arch.Align()
	// TODO: allow setting brk addr for raw Usercorn?
	if u.brk > 0 {
		mask := uint64(4096)
		u.brk = (u.brk + mask) & ^(mask - 1)
	}
	// make sure PC is set to entry point for debuggers
	u.RegWrite(u.Arch().PC, u.Entry())
	return err
}

// Fs returns the filesystem
func (u *Usercorn) Fs() *ramfs.Filesystem {
	return u.fs
}

// ------------------ everything below is old code

// HookAdd adds a CPU hook.
func (u *Usercorn) HookAdd(htype int, cb interface{}, begin, end uint64, extra ...int) (cpu.Hook, error) {
	hh, err := u.Cpu.HookAdd(htype, cb, begin, end, extra...)
	if err == nil {
		u.hooks = append(u.hooks, hh)
	}
	return hh, err
}

// HookDel removes the hook
func (u *Usercorn) HookDel(hh cpu.Hook) error {
	tmp := make([]cpu.Hook, 0, len(u.hooks))
	for _, v := range u.hooks {
		if v != hh {
			tmp = append(tmp, v)
		}
	}
	u.hooks = tmp
	return u.Cpu.HookDel(hh)
}

func (u *Usercorn) Run() error {
	defer func() {
		for _, v := range u.hooks {
			u.HookDel(v)
		}
		if e := recover(); e != nil {
			fmt.Printf("\n+++ caught panic +++\n%s\n%s\n\n", e, debug.Stack())
			panic(e)
		}
	}()
	// TODO: hooks are removed below but if Run() is called again the OS stack will be reinitialized
	// maybe won't be a problem if the stack is zeroed and stack pointer is reset?
	// or OS stack init can be moved somewhere else (like NewUsercorn)
	if u.os.Init != nil {
		if err := u.os.Init(u, u.config.Args, u.config.Env); err != nil {
			return err
		}
	}
	if err := u.addHooks(); err != nil {
		return err
	}
	// in case this isn't the first run
	u.exitStatus = nil
	// loop to restart Cpu if we need to call a trampoline function
	u.RegWrite(u.arch.PC, u.entry)
	var err error
	for err == nil && u.exitStatus == nil {
		pc, _ := u.RegRead(u.arch.PC)
		err = u.Start(pc, u.exit)

		if u.restart != nil {
			err = u.restart(u, err)
			u.restart = nil
			if err != nil {
				break
			}
		}
		pc, _ = u.RegRead(u.arch.PC)
		if len(u.trampolines) > 0 {
			sp, _ := u.RegRead(u.arch.SP)
			trampolines := u.trampolines
			u.trampolines = nil
			// TODO: trampolines should be annotated in trace
			// trampolines should show up during symbolication?
			// FIXME: binary tracer does NOT handle this yet
			u.trampolined = true
			for _, tramp := range trampolines {
				if err = tramp.fun(); err != nil {
					break
				}
			}
			u.trampolined = false
			u.RegWrite(u.arch.PC, pc)
			u.RegWrite(u.arch.SP, sp)
		}
	}
	if _, ok := err.(models.ExitStatus); !ok && err != nil || u.config.Verbose {
		if err != nil {
			fmt.Printf("got error: %v", err)
		}
	}
	if err == nil && u.exitStatus != nil {
		err = u.exitStatus
	}
	return err
}

func (u *Usercorn) Start(pc, end uint64) error {
	u.running = true
	err := u.Cpu.Start(pc, end)
	u.running = false
	return err
}

func (u *Usercorn) Exe() string {
	return u.exe
}

func (u *Usercorn) Loader() models.Loader {
	return u.loader
}

func (u *Usercorn) Base() uint64 {
	// points to program base
	return u.base
}

func (u *Usercorn) Entry() uint64 {
	// points to effective program entry: either an interpreter or the binary
	return u.entry
}

func (u *Usercorn) SetEntry(entry uint64) {
	u.entry = entry
}

func (u *Usercorn) SetExit(exit uint64) {
	u.exit = exit
}

func (u *Usercorn) BinEntry() uint64 {
	// points to binary entry, even if an interpreter is used
	return u.binEntry
}

func (u *Usercorn) Brk(addr uint64) (uint64, error) {
	// TODO: brk(0) behavior is linux specific
	cur := u.brk
	if addr > 0 && addr >= cur {
		// take brk protections from last brk segment (not sure if this is right)
		prot := cpu.PROT_READ | cpu.PROT_WRITE
		if brk := u.memsim.Mem.Find(cur); brk != nil {
			prot = brk.Prot
			u.brk = brk.Addr + brk.Size
		}
		size := addr - u.brk
		if size > 0 {
			_, err := u.Mmap(u.brk, size, prot, true, "brk", nil)
			if err != nil {
				return u.brk, err
			}
		}
		u.brk = addr
	}
	return u.brk, nil
}

func (u *Usercorn) addHooks() error {
	// TODO: this sort of error should be handled in ui module?
	// issue #244
	invalid := cpu.HOOK_MEM_ERR
	u.HookAdd(invalid, func(_ cpu.Cpu, access int, addr uint64, size int, value int64) bool {
		switch access {
		case cpu.MEM_WRITE_UNMAPPED, cpu.MEM_WRITE_PROT:
			fmt.Printf("invalid write")
		case cpu.MEM_READ_UNMAPPED, cpu.MEM_READ_PROT:
			fmt.Printf("invalid read")
		case cpu.MEM_FETCH_UNMAPPED, cpu.MEM_FETCH_PROT:
			fmt.Printf("invalid fetch")
		default:
			fmt.Printf("unknown memory error")
		}
		fmt.Printf(": @0x%x, 0x%x = 0x%x\n", addr, size, uint64(value))
		return false
	}, 1, 0)
	u.HookAdd(cpu.HOOK_INTR, func(_ cpu.Cpu, intno uint32) {
		u.os.Interrupt(u, intno)
	}, 1, 0)
	return nil
}

func (u *Usercorn) mapBinary(f *os.File) (entry, base, realEntry uint64, err error) {
	l := u.loader
	var dynamic bool
	switch l.Type() {
	case loader.EXEC:
		dynamic = false
	case loader.DYN:
		dynamic = true
	default:
		err = errors.New("unsupported file load type")
		return
	}
	// find segment bounds
	segments, err := l.Segments()
	if err != nil {
		return
	}
	low := ^uint64(0)
	high := uint64(0)
	for _, seg := range segments {
		if seg.Size == 0 {
			continue
		}
		if seg.Addr < low {
			low = seg.Addr
		}
		h := seg.Addr + seg.Size
		if h > high {
			high = h
		}
	}
	if low > high {
		low = high
	}
	// map contiguous binary
	loadBias := uint64(0)
	if dynamic {
		mapLow := low
		if loadBias > 0 {
			mapLow = loadBias
		} else if mapLow == 0 {
			mapLow = 0x1000000
		}
		// TODO: is allocating the whole lib width remotely sane?
		var page *cpu.Page
		page, err = u.MemReserve(mapLow, high-low, false)
		if err != nil {
			return
		}
		loadBias = page.Addr - low
	}
	desc := "exe"

	// initial forced segment mappings
	for _, seg := range segments {
		prot := seg.Prot
		if prot == 0 {
			// TODO: confirm why darwin needs this
			prot = cpu.PROT_ALL
		}
		fileDesc := &cpu.FileDesc{Name: f.Name(), Off: seg.Off, Len: seg.Size}
		_, err = u.Mmap(loadBias+seg.Addr, seg.Size, prot, true, desc, fileDesc)
		if err != nil {
			return
		}
	}
	// merge overlapping segments when writing contents to memory
	merged := make([]*models.Segment, 0, len(segments))
outer:
	for _, seg := range segments {
		if seg.Size == 0 {
			continue
		}
		addr, size := align(seg.Addr, seg.Size, true)
		s := &models.Segment{Start: addr, End: addr + size, Prot: seg.Prot}
		for _, s2 := range merged {
			if s2.Overlaps(s) {
				s2.Merge(s)
				continue outer
			}
		}
		merged = append(merged, s)
	}
	// write segment memory
	mem := u.Mem()
	var data []byte
	for _, seg := range segments {
		if data, err = seg.Data(); err != nil {
			return
		}
		mem.Seek(int64(loadBias+seg.Addr), io.SeekStart)
		if _, err = mem.Write(data); err != nil {
			return
		}
	}
	entry = loadBias + l.Entry()
	// load interpreter if present
	return entry, loadBias, entry, nil
}

func (u *Usercorn) MapStack(base, size uint64, guard bool) error {
	u.StackBase = base
	u.StackSize = size
	// TODO: check for NX stack?
	addr, err := u.Mmap(base, size, cpu.PROT_ALL, true, "stack", nil)
	if err != nil {
		return err
	}
	stackEnd := addr + size
	if err := u.RegWrite(u.arch.SP, stackEnd); err != nil {
		return err
	}
	if guard {
		_, err := u.Mmap(stackEnd, UC_MEM_ALIGN, cpu.PROT_NONE, true, "stack guard", nil)
		return err
	}
	return nil
}

func (u *Usercorn) AddKernel(kernel interface{}, first bool) {
	kco := kernel.(co.Kernel)
	if first {
		u.kernels = append([]co.Kernel{kco}, u.kernels...)
	} else {
		u.kernels = append(u.kernels, kco)
	}
}

func (u *Usercorn) Kernel(i int) interface{} {
	return u.kernels[i]
}

func (u *Usercorn) Syscall(num int, name string, getArgs models.SysGetArgs) (uint64, error) {
	if name == "" {
		msg := fmt.Sprintf("Syscall missing: %d", num)
		panic(msg)
	}
	for _, k := range u.kernels {
		if sys := co.Lookup(u, k, name); sys != nil {
			args, err := getArgs(len(sys.In))
			if err != nil {
				return 0, err
			}
			ret := sys.Call(args)
			return ret, nil
		}
	}
	msg := errors.Errorf("Kernel not found for syscall '%s'", name)
	panic(msg)
}

func (u *Usercorn) Exit(err error) {
	u.exitStatus = err
	u.Stop()
}

func (u *Usercorn) Close() error {
	var err error
	u.final.Do(func() {
		err = u.Cpu.Close()
	})
	return err
}

func (u *Usercorn) StrucAt(addr uint64) *models.StrucStream {
	options := &struc.Options{
		Order:   u.ByteOrder(),
		PtrSize: int(u.Bits()),
	}
	r := u.Mem()
	r.Seek(int64(addr), io.SeekStart)
	return models.NewStrucStream(r, options)
}

func (u *Usercorn) Config() *models.Config { return u.config }

func (u *Usercorn) Restart(fn func(models.Usercorn, error) error) {
	u.restart = fn
	u.Stop()
}

func (u *Usercorn) Trampoline(fun func() error) error {
	if u.running {
		desc := ""
		if _, file, line, ok := runtime.Caller(1); ok {
			desc = fmt.Sprintf("%s:%d", file, line)
		}
		u.trampolines = append(u.trampolines, tramp{
			desc: desc,
			fun:  fun,
		})
		return u.Stop()
	}
	return fun()
}
