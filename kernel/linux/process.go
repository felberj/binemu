package linux

import "github.com/felberj/binemu/models"

// Exit sycall
func (k *LinuxKernel) Exit(code uint64) {
	k.U.Exit(models.ExitStatus(code))
}

// ExitGroup syscall
func (k *LinuxKernel) ExitGroup(code uint64) {
	k.Exit(code)
}

// Ugetrlimit syscall (not implemented)
func (k *LinuxKernel) Ugetrlimit() {}

// Getrlimit syscall (not implemented)
func (k *LinuxKernel) Getrlimit() {}
