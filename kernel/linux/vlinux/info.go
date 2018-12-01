package vlinux

import (
	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/models"
)

// Uname syscall
func (k *VirtualLinuxKernel) Uname(buf co.Buf) {
	uname := &models.Uname{
		Sysname:  "Linux",
		Nodename: "usercorn",
		Release:  "3.13.0-24-generic",
		Version:  "normal copy of Linux minding my business",
		Machine:  k.U.Loader().Arch(),
	}
	// Pad is both OS and arch dependent? :(
	uname.Pad(65)
	buf.Pack(uname)
}
