package sparc

import (
	"fmt"

	uc "github.com/felberj/binemu/unicorn"
	"github.com/lunixbochs/ghostrace/ghost/sys/num"

	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/kernel/linux"
	"github.com/felberj/binemu/models"
)

// TODO: sparc linux syscall abi regs
var LinuxRegs = []int{}

type LinuxKernel struct {
	*linux.LinuxKernel
}

func LinuxKernels(u models.Usercorn) []interface{} {
	kernel := &LinuxKernel{linux.NewKernel(u.Fs())}
	return []interface{}{kernel}
}

func LinuxSyscall(u models.Usercorn) {
	// TODO: handle errors or something
	g1, _ := u.RegRead(uc.SPARC_REG_G1)
	// TODO: add sparc x86 syscall numbers to ghostrace
	name, _ := num.Linux_x86[int(g1)]
	ret, _ := u.Syscall(int(g1), name, co.RegArgs(u, LinuxRegs))
	u.RegWrite(uc.SPARC_REG_O0, ret)
}

// TODO: add sparc syscall convention support
func LinuxInterrupt(u models.Usercorn, intno uint32) {
	panic(fmt.Sprintf("unknown interrupt: %d", intno))
}

func init() {
	Arch.RegisterOS(&models.OS{
		Name:      "linux",
		Kernels:   LinuxKernels,
		Init:      linux.StackInit,
		Interrupt: LinuxInterrupt,
	})
}
