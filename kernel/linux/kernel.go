package linux

import (
	"github.com/felberj/binemu/kernel/posix"
	"github.com/felberj/binemu/models"
)

const (
	STACK_BASE = 0xbf800000
	STACK_SIZE = 0x00800000
)

type LinuxKernel struct {
	posix.PosixKernel
}

func NewKernel() *LinuxKernel {
	kernel := &LinuxKernel{*posix.NewKernel()}
	registerUnpack(kernel)
	kernel.Pack = Pack
	return kernel
}

func StackInit(u models.Usercorn, args, env []string) error {
	if err := u.MapStack(STACK_BASE, STACK_SIZE, false); err != nil {
		return err
	}
	auxv, err := SetupElfAuxv(u)
	if err != nil {
		return err
	}
	return posix.StackInit(u, args, env, auxv)
}
