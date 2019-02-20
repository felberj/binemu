package x86_16

import (
	"github.com/felberj/binemu/cpu"
	"github.com/felberj/binemu/models"

	uc "github.com/felberj/binemu/cpu/unicorn"
)

var Arch = &models.Arch{
	Name: "x86_16",
	Bits: 16,

	Cpu: &cpu.Builder{Arch: uc.ARCH_X86, Mode: uc.MODE_16},

	PC: uc.X86_REG_IP,
	SP: uc.X86_REG_SP,
	Regs: map[string]int{
		"ip": uc.X86_REG_IP,
		"sp": uc.X86_REG_SP,
		"bp": uc.X86_REG_BP,
		"ax": uc.X86_REG_AX,
		"bx": uc.X86_REG_BX,
		"cx": uc.X86_REG_CX,
		"dx": uc.X86_REG_DX,
		"si": uc.X86_REG_SI,
		"di": uc.X86_REG_DI,

		"flags": uc.X86_REG_EFLAGS,

		"cs": uc.X86_REG_CS,
		"ds": uc.X86_REG_DS,
		"es": uc.X86_REG_ES,
		"ss": uc.X86_REG_SS,
	},
	DefaultRegs: []string{
		"ax", "bx", "cx", "dx", "si", "di", "bp",
	},
}
