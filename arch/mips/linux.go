package mips

import (
	"fmt"

	uc "github.com/felberj/binemu/cpu/unicorn"
	sysnum "github.com/lunixbochs/ghostrace/ghost/sys/num"

	"github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/kernel/linux"
	"github.com/felberj/binemu/models"
)

var LinuxRegs = []int{uc.MIPS_REG_A0, uc.MIPS_REG_A1, uc.MIPS_REG_A2, uc.MIPS_REG_A3, 0, 0}

type MipsLinuxKernel struct {
	*linux.LinuxKernel
}

func (k *MipsLinuxKernel) SetThreadArea(addr uint64) error {
	panic("not implemented")
}

func LinuxKernels(u models.Usercorn) []interface{} {
	kernel := &MipsLinuxKernel{LinuxKernel: linux.NewKernel(u.Fs())}
	return []interface{}{kernel}
}

func LinuxInit(u models.Usercorn, args, env []string) error {
	return linux.StackInit(u, args, env)
}

func LinuxSyscall(u models.Usercorn) {
	// TODO: handle errors or something
	num, _ := u.RegRead(uc.MIPS_REG_V0)
	name, _ := sysnum.Linux_mips[int(num)]
	ret, _ := u.Syscall(int(num), name, common.RegArgs(u, LinuxRegs))
	u.RegWrite(uc.MIPS_REG_V0, ret)
}

func LinuxInterrupt(u models.Usercorn, cause uint32) {
	intno := (cause >> 1) & 15
	if intno == 8 {
		LinuxSyscall(u)
		return
	}
	panic(fmt.Sprintf("unhandled MIPS interrupt %d", intno))
}

func init() {
	Arch.RegisterOS(&models.OS{
		Name:      "linux",
		Kernels:   LinuxKernels,
		Init:      LinuxInit,
		Interrupt: LinuxInterrupt,
	})
}
