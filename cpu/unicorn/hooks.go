package unicorn

import (
	"github.com/felberj/binemu/models/cpu"
	"github.com/pkg/errors"

	uc "github.com/felberj/binemu/unicorn"
)

// CodeHook is used to hook executed code
type CodeHook func(addr uint64, size uint32)

// MemHook is used to hook memory access (read/write)
type MemHook func(access int, addr uint64, size int, val int64)

// MemErrHook is used to hook memory access errors
type MemErrHook func(access int, addr uint64, size int, val int64) bool

// InterruptHook is used to hook interrupts
type InterruptHook func(intno uint32)

// InstructionHook is used to hook instructions
type InstructionHook func()

// HookBlock TODO figure out what it does
func (u *Cpu) HookBlock(cb CodeHook, start, end uint64) (uc.Hook, error) {
	wrap := func(_ *uc.Unicorn, addr uint64, size uint32) { cb(addr, size) }
	return u.Unicorn.HookAdd(cpu.HOOK_BLOCK, wrap, start, end)
}

// HookCode TODO figure out what it does
func (u *Cpu) HookCode(cb CodeHook, start, end uint64) (uc.Hook, error) {
	wrap := func(_ *uc.Unicorn, addr uint64, size uint32) { cb(addr, size) }
	return u.Unicorn.HookAdd(cpu.HOOK_CODE, wrap, start, end)
}

// HookMem TODO figure out what it does
func (u *Cpu) HookMem(kind int, cb MemHook, start, end uint64) (uc.Hook, error) {
	kind &= cpu.HOOK_MEM_READ | cpu.HOOK_MEM_WRITE
	if kind == 0 {
		return 0, errors.New("invalid hook type")
	}
	wrap := func(_ *uc.Unicorn, access int, addr uint64, size int, val int64) { cb(access, addr, size, val) }
	return u.Unicorn.HookAdd(cpu.HOOK_MEM_READ, wrap, start, end)
}

// HookMemErr TODO figure out what it does
func (u *Cpu) HookMemErr(kind int, cb MemErrHook, start, end uint64) (uc.Hook, error) {
	kind &= uc.HOOK_MEM_READ_UNMAPPED | uc.HOOK_MEM_WRITE_UNMAPPED | uc.HOOK_MEM_FETCH_UNMAPPED |
		uc.HOOK_MEM_READ_PROT | uc.HOOK_MEM_WRITE_PROT | uc.HOOK_MEM_FETCH_PROT
	if kind == 0 {
		return 0, errors.New("invalid hook type")
	}
	wrap := func(_ *uc.Unicorn, access int, addr uint64, size int, val int64) bool {
		return cb(access, addr, size, val)
	}
	return u.Unicorn.HookAdd(cpu.HOOK_MEM_READ, wrap, start, end)
}

// HookInterrupt TODO figure out what it does
func (u *Cpu) HookInterrupt(cb InterruptHook, start, end uint64) (uc.Hook, error) {
	wrap := func(_ *uc.Unicorn, intno uint32) { cb(intno) }
	return u.Unicorn.HookAdd(cpu.HOOK_INTR, wrap, start, end)
}

// HookInstruction TODO figure out what it does
func (u *Cpu) HookInstruction(cb InstructionHook, start, end uint64, instruction int) (uc.Hook, error) {
	wrap := func(_ *uc.Unicorn) { cb() }
	return u.Unicorn.HookAdd(cpu.HOOK_INSN, wrap, start, end, instruction)
}
