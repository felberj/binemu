package arm

import (
	uc "github.com/felberj/binemu/unicorn"
	cs "github.com/lunixbochs/capstr"

	"github.com/felberj/binemu/cpu"
	"github.com/felberj/binemu/cpu/unicorn"
	"github.com/felberj/binemu/models"
)

var Arch = &models.Arch{
	Name:   "arm",
	Bits:   32,
	Radare: "arm",

	Cpu: &unicorn.Builder{Arch: uc.ARCH_ARM, Mode: uc.MODE_ARM},
	Dis: &cpu.Capstr{Arch: cs.ARCH_ARM, Mode: cs.MODE_ARM},

	PC: uc.ARM_REG_PC,
	SP: uc.ARM_REG_SP,
	Regs: map[string]int{
		"r0":  uc.ARM_REG_R0,
		"r1":  uc.ARM_REG_R1,
		"r2":  uc.ARM_REG_R2,
		"r3":  uc.ARM_REG_R3,
		"r4":  uc.ARM_REG_R4,
		"r5":  uc.ARM_REG_R5,
		"r6":  uc.ARM_REG_R6,
		"r7":  uc.ARM_REG_R7,
		"r8":  uc.ARM_REG_R8,
		"r9":  uc.ARM_REG_R9,
		"r10": uc.ARM_REG_R10,
		"r11": uc.ARM_REG_R11,
		"r12": uc.ARM_REG_R12,
		"lr":  uc.ARM_REG_LR,
		"sp":  uc.ARM_REG_SP,
		"pc":  uc.ARM_REG_PC,
	},
	DefaultRegs: []string{
		"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7", "r8",
		"r9", "r10", "r11", "r12",
	},
}

func EnterUsermode(u models.Usercorn) error {
	panic("not implemented")
}

func EnableFPU(u models.Usercorn) error {
	val, err := u.RegRead(uc.ARM_REG_C1_C0_2)
	if err != nil {
		return err
	}
	if err = u.RegWrite(uc.ARM_REG_C1_C0_2, val|(0xf<<20)); err != nil {
		return err
	}
	return u.RegWrite(uc.ARM_REG_FPEXC, 0x40000000)
}
