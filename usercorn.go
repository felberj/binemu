package usercorn

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	rdebug "runtime/debug"
	"sync"

	"github.com/felberj/binemu/arch"
	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/loader"
	"github.com/felberj/binemu/models"
	"github.com/felberj/binemu/models/cpu"
	"github.com/felberj/binemu/models/debug"
	"github.com/felberj/binemu/models/trace"
	"github.com/felberj/ramfs"
	"github.com/lunixbochs/ghostrace/ghost/memio"
	"github.com/lunixbochs/struc"
	"github.com/pkg/errors"
)

// #cgo LDFLAGS: -Wl,-rpath,\$ORIGIN/deps/lib:\$ORIGIN/lib
import "C"

type tramp struct {
	desc string
	fun  func() error
}

type Usercorn struct {
	*Task

	sync.Mutex
	config       *models.Config
	exe          string
	loader       models.Loader
	interpLoader models.Loader
	kernels      []co.Kernel
	memio        memio.MemIO

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

	gate models.Gate

	breaks       []*models.Breakpoint
	futureBreaks []*models.Breakpoint

	hooks    []cpu.Hook
	sysHooks []*models.SysHook
	fs       *ramfs.Filesystem

	debug    *debug.Debug
	trace    *trace.Trace
	replay   *trace.Replay
	rewind   []models.Op
	inscount uint64
}

// NewUsercornWrapper is just a hacky woraround that usercorn has privat fields.
// TODO(felberj) remove
func NewUsercornWrapper(exe string, t *Task, fs *ramfs.Filesystem, l models.Loader, os *models.OS, c *models.Config) *Usercorn {
	u := &Usercorn{
		Task:   t,
		config: c,
		loader: l,
		exit:   0xffffffffffffffff,
		debug:  debug.NewDebug(l.Arch(), c),
		fs:     fs,
	}
	u.memio = memio.NewMemIO(
		// ReadAt() callback
		func(p []byte, addr uint64) (int, error) {
			if err := u.Task.MemReadInto(p, addr); err != nil {
				return 0, err
			}
			return len(p), nil
		},
		// WriteAt() callback
		func(p []byte, addr uint64) (int, error) {
			if err := u.Task.MemWrite(addr, p); err != nil {
				return 0, err
			}
			return len(p), nil
		},
	)
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
	u.interpBase, u.entry, u.base, u.binEntry, err = u.mapBinary(f, false)
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

func NewUsercornRaw(l models.Loader, config *models.Config) (*Usercorn, error) {
	config = config.Init()

	a, OS, err := arch.GetArch(l.Arch(), l.OS())
	if err != nil {
		return nil, err
	}
	cpu, err := a.Cpu.New()
	if err != nil {
		return nil, err
	}
	task := NewTask(cpu, a, OS, l.ByteOrder())
	u := &Usercorn{
		Task:   task,
		config: config,
		loader: l,
		exit:   0xffffffffffffffff,
		debug:  debug.NewDebug(l.Arch(), config),
	}
	if u.config.Rewind {
		u.replay = trace.NewReplay(u.arch, u.os, l.ByteOrder(), u.debug)
		config.Trace.OpCallback = append(config.Trace.OpCallback, u.replay.Feed)
		if u.config.Rewind {
			u.rewind = make([]models.Op, 0, 10000)
			config.Trace.OpCallback = append(config.Trace.OpCallback,
				func(frame models.Op) {
					u.rewind = append(u.rewind, frame)
				})
		}
	}
	u.trace, err = trace.NewTrace(u, &config.Trace)
	if err != nil {
		return nil, errors.Wrap(err, "NewTrace() failed")
	}
	u.memio = memio.NewMemIO(
		// ReadAt() callback
		func(p []byte, addr uint64) (int, error) {
			if err := u.Task.MemReadInto(p, addr); err != nil {
				return 0, err
			}
			if u.trace != nil && u.config.Trace.Mem {
				u.trace.OnMemReadSize(addr, uint32(len(p)))
			}
			return len(p), nil
		},
		// WriteAt() callback
		func(p []byte, addr uint64) (int, error) {
			if err := u.Task.MemWrite(addr, p); err != nil {
				return 0, err
			}
			if u.trace != nil && u.config.Trace.Mem {
				u.trace.OnMemWrite(addr, p)
			}
			return len(p), nil
		},
	)
	// load kernels
	// the array cast is a trick to work around circular imports
	if OS.Kernels != nil {
		kernelI := OS.Kernels(u)
		kernels := make([]co.Kernel, len(kernelI))
		for i, k := range kernelI {
			kernels[i] = k.(co.Kernel)
		}
		u.kernels = kernels
	}
	return u, nil
}

func NewUsercorn(exe string, config *models.Config) (models.Usercorn, error) {
	config = config.Init()

	f, err := os.Open(exe)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	l, err := loader.LoadArch(f, "any", config.OSHint)
	if err != nil {
		return nil, err
	}
	u, err := NewUsercornRaw(l, config)
	if err != nil {
		return nil, err
	}
	exe, _ = filepath.Abs(exe)
	u.exe = exe
	u.loader = l

	// map binary (and interp) into memory
	u.interpBase, u.entry, u.base, u.binEntry, err = u.mapBinary(f, false)
	if err != nil {
		return nil, err
	}
	// find data segment for brk
	u.brk = 0
	segments, err := l.Segments()
	if err != nil {
		return nil, err
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
	return u, nil
}

func (u *Usercorn) Callstack() []models.Stackframe {
	if u.replay == nil {
		return nil
	}
	return u.replay.Callstack.Freeze(u.replay.PC, u.replay.SP)
}

func (u *Usercorn) Rewind(by, addr uint64) error {
	panic("not implemented")
}

// Intercept memory read/write into MemIO to make tracing always work.
// This means Trace needs to use Task().Read() instead
func (u *Usercorn) MemWrite(addr uint64, p []byte) error {
	_, err := u.memio.WriteAt(p, addr)
	return err
}
func (u *Usercorn) MemReadInto(p []byte, addr uint64) error {
	_, err := u.memio.ReadAt(p, addr)
	return err
}
func (u *Usercorn) MemRead(addr, size uint64) ([]byte, error) {
	p := make([]byte, size)
	err := u.MemReadInto(p, addr)
	return p, err
}

// read without tracing, used by trace and repl
func (u *Usercorn) DirectRead(addr, size uint64) ([]byte, error) {
	return u.Task.MemRead(addr, size)
}
func (u *Usercorn) DirectWrite(addr uint64, p []byte) error {
	return u.Task.MemWrite(addr, p)
}

func (u *Usercorn) HookAdd(htype int, cb interface{}, begin, end uint64, extra ...int) (cpu.Hook, error) {
	hh, err := u.Cpu.HookAdd(htype, cb, begin, end, extra...)
	if err == nil {
		u.hooks = append(u.hooks, hh)
	}
	return hh, err
}

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

func (u *Usercorn) HookSysAdd(before, after models.SysCb) *models.SysHook {
	hook := &models.SysHook{Before: before, After: after}
	u.sysHooks = append(u.sysHooks, hook)
	return hook
}

func (u *Usercorn) HookSysDel(hook *models.SysHook) {
	tmp := make([]*models.SysHook, 0, len(u.sysHooks)-1)
	for _, v := range u.sysHooks {
		if v != hook {
			tmp = append(tmp, v)
		}
	}
	u.sysHooks = tmp
}

func (u *Usercorn) Run() error {
	// TODO: defers are expensive I hear
	defer func() {
		for _, v := range u.hooks {
			u.HookDel(v)
		}
		if e := recover(); e != nil {
			fmt.Printf("\n+++ caught panic +++\n%s\n%s\n\n", e, rdebug.Stack())
			panic(e)
		}
	}()
	// PrefixArgs was added for shebang
	if len(u.config.PrefixArgs) > 0 {
		u.config.Args = append(u.config.PrefixArgs, u.config.Args...)
	}
	// TODO: hooks are removed below but if Run() is called again the OS stack will be reinitialized
	// maybe won't be a problem if the stack is zeroed and stack pointer is reset?
	// or OS stack init can be moved somewhere else (like NewUsercorn)
	if u.os.Init != nil {
		if err := u.os.Init(u, u.config.Args, u.config.Env); err != nil {
			return err
		}
	}
	if u.config.Trace.Any() {
		if err := u.trace.Attach(); err != nil {
			return err
		}
	}
	if err := u.addHooks(); err != nil {
		return err
	}
	if u.config.InsCount {
		u.HookAdd(cpu.HOOK_CODE, func(_ cpu.Cpu, addr uint64, size uint32) {
			u.inscount++
		}, 1, 0)
	}
	// in case this isn't the first run
	u.exitStatus = nil
	// loop to restart Cpu if we need to call a trampoline function
	u.RegWrite(u.arch.PC, u.entry)
	var err error
	for err == nil && u.exitStatus == nil {
		// well there's a huge pile of sync here to make sure everyone's ready to go...
		u.gate.Start()
		// allow a repl to break us out with u.Exit() before we run
		if u.exitStatus != nil {
			break
		}
		// allow repl or rewind to change pc
		pc, _ := u.RegRead(u.arch.PC)
		err = u.Start(pc, u.exit)
		u.gate.Stop()

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
	if u.config.InsCount {
		u.Printf("inscount: %d\n", u.inscount)
	}
	if err == nil && u.exitStatus != nil {
		err = u.exitStatus
	}
	return err
}

func (u *Usercorn) Gate() *models.Gate {
	return &u.gate
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

func (u *Usercorn) InterpBase() uint64 {
	// points to interpreter base or 0
	return u.interpBase
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

func (u *Usercorn) PrefixPath(path string, force bool) string {
	return u.config.PrefixPath(path, force)
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
			u.Printf("invalid write")
		case cpu.MEM_READ_UNMAPPED, cpu.MEM_READ_PROT:
			u.Printf("invalid read")
		case cpu.MEM_FETCH_UNMAPPED, cpu.MEM_FETCH_PROT:
			u.Printf("invalid fetch")
		default:
			u.Printf("unknown memory error")
		}
		u.Printf(": @0x%x, 0x%x = 0x%x\n", addr, size, uint64(value))
		return false
	}, 1, 0)
	u.HookAdd(cpu.HOOK_INTR, func(_ cpu.Cpu, intno uint32) {
		u.os.Interrupt(u, intno)
	}, 1, 0)
	return nil
}

func (u *Usercorn) mapBinary(f *os.File, isInterp bool) (interpBase, entry, base, realEntry uint64, err error) {
	l := u.loader
	if isInterp {
		l, err = loader.LoadArch(f, l.Arch(), l.OS())
		if err != nil {
			return
		}
		u.interpLoader = l
	}
	var dynamic bool
	switch l.Type() {
	case loader.EXEC:
		dynamic = false
	case loader.DYN:
		dynamic = true
	default:
		err = errors.New("Unsupported file load type.")
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
	loadBias := u.config.ForceBase
	if isInterp {
		loadBias = u.config.ForceInterpBase
		// reserve space at end of bin for brk
		barrier := u.brk + 8*1024*1024
		if loadBias <= barrier {
			loadBias = barrier
		}
	}
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
	var desc string
	if isInterp {
		desc = "interp"
	} else {
		desc = "exe"
	}
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
	var data []byte
	for _, seg := range segments {
		if data, err = seg.Data(); err != nil {
			return
		}
		if err = u.MemWrite(loadBias+seg.Addr, data); err != nil {
			return
		}
	}
	entry = loadBias + l.Entry()
	// load interpreter if present
	interp := l.Interp()
	if interp != "" && !isInterp && !u.config.SkipInterp {
		f, err = os.Open(u.PrefixPath(interp, true))
		if err != nil {
			return
		}
		defer f.Close()
		u.brk = high
		var interpBias, interpEntry uint64
		_, _, interpBias, interpEntry, err = u.mapBinary(f, true)
		if u.interpLoader.Arch() != l.Arch() {
			err = errors.Errorf("Interpreter arch mismatch: %s != %s", l.Arch(), u.interpLoader.Arch())
			return
		}
		return interpBias, interpEntry, loadBias, entry, err
	} else {
		return 0, entry, loadBias, entry, nil
	}
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
		if u.config.StubSyscalls {
			u.Println(msg)
		} else {
			panic(msg)
		}
	}
	if u.config.BlockSyscalls {
		return 0, nil
	}
	for _, k := range u.kernels {
		if sys := co.Lookup(u, k, name); sys != nil {
			args, err := getArgs(len(sys.In))
			if err != nil {
				return 0, err
			}
			desc := sys.Trace(args)
			prevent := false
			for _, v := range u.sysHooks {
				if v.Before(num, name, args, 0, desc) {
					prevent = true
				}
			}
			if prevent {
				return 0, nil
			}
			ret := sys.Call(args)
			desc = sys.TraceRet(args, ret)
			for _, v := range u.sysHooks {
				v.After(num, name, args, ret, desc)
			}
			return ret, nil
		}
	}
	// TODO: hook unknown syscalls?
	msg := errors.Errorf("Kernel not found for syscall '%s'", name)
	if u.config.StubSyscalls {
		u.Println(msg)
		return 0, nil
	} else {
		panic(msg)
	}
}

func (u *Usercorn) Exit(err error) {
	u.exitStatus = err
	u.Stop()
	if u.trace != nil {
		u.trace.OnExit()
	}
}

func (u *Usercorn) Close() error {
	var err error
	u.final.Do(func() {
		err = u.Cpu.Close()
	})
	return err
}

func (u *Usercorn) Mem() memio.MemIO {
	return u.memio
}

func (u *Usercorn) StrucAt(addr uint64) *models.StrucStream {
	options := &struc.Options{
		Order:   u.ByteOrder(),
		PtrSize: int(u.Bits()),
	}
	return models.NewStrucStream(u.Mem().StreamAt(addr), options)
}

func (u *Usercorn) Config() *models.Config { return u.config }

func (u *Usercorn) Printf(f string, args ...interface{}) { fmt.Fprintf(u.config.Output, f, args...) }
func (u *Usercorn) Println(s ...interface{})             { fmt.Fprintln(u.config.Output, s...) }

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
	} else {
		return fun()
	}
}

// like RunShellcode but you're expected to map memory yourself
func (u *Usercorn) RunShellcodeMapped(addr uint64, code []byte, setRegs map[int]uint64, regsClobbered []int) error {
	return u.Trampoline(func() error {
		if regsClobbered == nil {
			regsClobbered = make([]int, len(setRegs))
			pos := 0
			for reg := range setRegs {
				regsClobbered[pos] = reg
				pos++
			}
		}
		// save clobbered regs
		savedRegs := make([]uint64, len(regsClobbered))
		for i, reg := range regsClobbered {
			savedRegs[i], _ = u.RegRead(reg)
		}
		// defer restoring saved regs
		defer func() {
			for i, reg := range regsClobbered {
				u.RegWrite(reg, savedRegs[i])
			}
		}()
		// set setRegs
		for reg, val := range setRegs {
			u.RegWrite(reg, val)
		}
		if err := u.MemWrite(addr, code); err != nil {
			return err
		}
		return u.Start(addr, addr+uint64(len(code)))
	})
}

// maps and runs shellcode at addr
// if regsClobbered is nil, setRegs will be saved/restored
// if addr is 0, we'll pick one for you
// if addr is already mapped, we will return an error
// so non-PIE is your problem
// will trampoline if unicorn is already running
func (u *Usercorn) RunShellcode(addr uint64, code []byte, setRegs map[int]uint64, regsClobbered []int) error {
	size := uint64(len(code))
	exists := len(u.memsim.Mem.FindRange(addr, size)) > 0
	if addr != 0 && exists {
		return errors.Errorf("RunShellcode: 0x%x - 0x%x overlaps mapped memory", addr, addr+uint64(len(code)))
	}
	mapped, err := u.Mmap(addr, size, cpu.PROT_ALL, true, "shellcode", nil)
	if err != nil {
		return err
	}
	defer u.Trampoline(func() error {
		return u.MemUnmap(mapped, size)
	})
	return u.RunShellcodeMapped(mapped, code, setRegs, regsClobbered)
}

var breakRe = regexp.MustCompile(`^((?P<addr>0x[0-9a-fA-F]+|\d+)|(?P<sym>[\w:]+(?P<off>\+0x[0-9a-fA-F]+|\d+)?)|(?P<source>.+):(?P<line>\d+))(@(?P<file>.+))?$`)

// adds a breakpoint to Usercorn instance
// see models.Breakpoint for desc syntax
// future=true adds it to the list of breakpoints to update when new memory is mapped/registered
func (u *Usercorn) BreakAdd(desc string, future bool, cb func(u models.Usercorn, addr uint64)) (*models.Breakpoint, error) {
	b, err := models.NewBreakpoint(desc, cb, u)
	if err != nil {
		return nil, err
	}
	u.breaks = append(u.breaks, b)
	if future {
		u.futureBreaks = append(u.futureBreaks, b)
	}
	return b, b.Apply()
}

// TODO: do these sort of operations while holding a lock?
func (u *Usercorn) BreakDel(b *models.Breakpoint) error {
	tmp := make([]*models.Breakpoint, 0, len(u.breaks))
	for _, v := range u.breaks {
		if v != b {
			tmp = append(tmp, v)
		}
	}
	u.breaks = tmp

	tmp = make([]*models.Breakpoint, 0, len(u.futureBreaks))
	for _, v := range u.futureBreaks {
		if v != b {
			tmp = append(tmp, v)
		}
	}
	u.futureBreaks = tmp

	return b.Remove()
}

func (u *Usercorn) Breakpoints() []*models.Breakpoint {
	return u.breaks
}

func (u *Usercorn) Symbolicate(addr uint64, includeSource bool) (*models.Symbol, string) {
	return u.debug.Symbolicate(addr, u.Task.Mappings(), includeSource)
}
