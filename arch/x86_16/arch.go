package x86_16

import (
	uc "github.com/felberj/binemu/unicorn"
	cs "github.com/lunixbochs/capstr"

	"github.com/felberj/binemu/cpu"
	"github.com/felberj/binemu/cpu/unicorn"
	"github.com/felberj/binemu/models"
)

var Arch = &models.Arch{
	Name:   "x86_16",
	Bits:   16,
	Radare: "x86",

	Cpu: &unicorn.Builder{Arch: uc.ARCH_X86, Mode: uc.MODE_16},
	Dis: &cpu.Capstr{Arch: cs.ARCH_X86, Mode: cs.MODE_16},

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
	GdbXml: "", // TODO: Find this GDB xml for x86_16
}
