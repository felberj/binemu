package mach

import (
	co "github.com/lunixbochs/usercorn/kernel/common"
)

type MachKernel struct {
	*co.KernelBase
}

func NewKernel() *MachKernel {
	return &MachKernel{&co.KernelBase{}}
}
