package models

import (
	"fmt"

	"github.com/felberj/binemu/cpu"

	uc "github.com/felberj/binemu/cpu/unicorn"
)

type CpuBuilder interface {
	New() (*cpu.Cpu, error)
}

type Reg struct {
	Enum    int
	Name    string
	Default bool
}

type RegVal struct {
	Reg
	Val uint64
}

type regList []Reg

type regMap map[string]int

func (r regMap) Items() regList {
	ret := make(regList, 0, len(r))
	for name, enum := range r {
		ret = append(ret, Reg{enum, name, false})
	}
	return ret
}

type Arch struct {
	Name string
	Bits int

	Cpu CpuBuilder

	PC   int
	SP   int
	OS   map[string]*OS
	Regs regMap

	DefaultRegs []string

	regNames map[int]string
	// sorted for RegDump
	regList  regList
	regEnums []int

	regBatch *uc.RegBatch
}

func (a *Arch) String() string {
	return fmt.Sprintf("<Arch %s>", a.Name)
}

func (a *Arch) RegNames() map[int]string {
	if a.regNames == nil {
		a.regNames = make(map[int]string, len(a.Regs))
		for name, enum := range a.Regs {
			a.regNames[enum] = name
		}
	}
	return a.regNames
}

func (a *Arch) RegisterOS(os *OS) {
	if a.OS == nil {
		a.OS = make(map[string]*OS)
	}
	if _, ok := a.OS[os.Name]; ok {
		panic("Duplicate OS " + os.Name)
	}
	a.OS[os.Name] = os
}

func (a *Arch) getRegList() regList {
	if a.regList == nil {
		rl := a.Regs.Items()
		for i, reg := range rl {
			// O(N) but it's a small list and only searched once
			for _, match := range a.DefaultRegs {
				if reg.Name == match {
					rl[i].Default = true
					break
				}
			}
		}
		a.regList = rl
	}
	return a.regList
}

func (a *Arch) RegEnums() []int {
	regList := a.getRegList()
	enums := make([]int, len(regList))
	for i, r := range regList {
		enums[i] = r.Enum
	}
	return enums
}

// FIXME: abstraction hack
func (a *Arch) RegDumpFast(c *cpu.Cpu) ([]uint64, error) {
	// manual check for Unicorn because Cpu interface doesn't have RegBatch for now
	if u, ok := c.Backend().(uc.Unicorn); ok {
		if a.regBatch == nil {
			var err error
			enums := a.RegEnums()
			a.regBatch, err = uc.NewRegBatch(enums)
			if err != nil {
				return nil, err
			}
		}
		return a.regBatch.ReadFast(u)
	} else {
		enums := a.RegEnums()
		out := make([]uint64, len(enums))
		for i, e := range enums {
			val, err := c.Unicorn.RegRead(e)
			if err != nil {
				return nil, err
			}
			out[i] = val
		}
		return out, nil
	}
}

func (a *Arch) RegDump(u *cpu.Cpu) ([]RegVal, error) {
	regList := a.getRegList()
	regs, err := a.RegDumpFast(u)
	if err != nil {
		return nil, err
	}
	ret := make([]RegVal, len(regList))
	for i, r := range regList {
		ret[i] = RegVal{r, regs[i]}
	}
	return ret, nil
}

type OS struct {
	Name      string
	Kernels   func(Usercorn) []interface{}
	Init      func(Usercorn, []string, []string) error
	Interrupt func(Usercorn, uint32)
}

func (o *OS) String() string {
	return fmt.Sprintf("<OS %s>", o.Name)
}
