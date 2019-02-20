package x86

import (
	"github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/models"
	"github.com/lunixbochs/ghostrace/ghost/sys/num"

	uc "github.com/felberj/binemu/cpu/unicorn"
)

const (
	dwMach = 1
	dwUnix = 2
	dwMdep = 3
	dwDiag = 4
)

func DarwinKernels(u models.Usercorn) []interface{} {
	return []interface{}{}
}

func DarwinInit(u models.Usercorn, args, env []string) error {
	return nil
}

func DarwinSyscall(u models.Usercorn, class int) {
	getArgs := common.StackArgs(u)

	eax, _ := u.RegRead(uc.X86_REG_EAX)
	nr := class<<24 | int(eax)
	name, _ := num.Darwin_x86_mach[nr]

	ret, _ := u.Syscall(nr, name, getArgs)
	u.RegWrite(uc.X86_REG_EAX, ret)
}

func DarwinInterrupt(u models.Usercorn, intno uint32) {
	switch intno {
	case 0x80:
		DarwinSyscall(u, dwUnix)
	case 0x81:
		DarwinSyscall(u, dwMach)
	case 0x82:
		DarwinSyscall(u, dwMdep)
	case 0x83:
		DarwinSyscall(u, dwDiag)
	}
}

func init() {
	Arch.RegisterOS(&models.OS{
		Name:      "darwin",
		Kernels:   DarwinKernels,
		Init:      DarwinInit,
		Interrupt: DarwinInterrupt,
	})
}
