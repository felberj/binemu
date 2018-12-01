package mach

import (
	co "github.com/felberj/binemu/kernel/common"
)

type MachKernel struct {
	*co.KernelBase
}

func NewKernel() *MachKernel {
	return &MachKernel{&co.KernelBase{}}
}
