package m68k

import (
	"github.com/felberj/binemu/cpu/unicorn"
	"github.com/felberj/binemu/models"

	uc "github.com/felberj/binemu/unicorn"
)

var Arch = &models.Arch{
	Name: "m68k",
	Bits: 32,

	Cpu: &unicorn.Builder{Arch: uc.ARCH_M68K, Mode: uc.MODE_BIG_ENDIAN},

	PC: uc.M68K_REG_PC,
	SP: uc.M68K_REG_A7,
	Regs: map[string]int{
		"d0": uc.M68K_REG_D0,
		"d1": uc.M68K_REG_D1,
		"d2": uc.M68K_REG_D2,
		"d3": uc.M68K_REG_D3,
		"d4": uc.M68K_REG_D4,
		"d5": uc.M68K_REG_D5,
		"d6": uc.M68K_REG_D6,
		"d7": uc.M68K_REG_D7,
		"a0": uc.M68K_REG_A0,
		"a1": uc.M68K_REG_A1,
		"a2": uc.M68K_REG_A2,
		"a3": uc.M68K_REG_A3,
		"a4": uc.M68K_REG_A4,
		"a5": uc.M68K_REG_A5,
		"a6": uc.M68K_REG_A6,
		"sp": uc.M68K_REG_A7,
		"pc": uc.M68K_REG_PC,
	},
	DefaultRegs: []string{
		"d0", "d1", "d2", "d3", "d4", "d5", "d6", "d7",
		"a0", "a1", "a2", "a3", "a4", "a5", "a6",
	},
}
