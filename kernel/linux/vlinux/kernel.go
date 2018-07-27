// Package vlinux provides a kernel that wants to separate the
// running process from the host system.
package vlinux

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/felberj/ramfs"
	co "github.com/lunixbochs/usercorn/kernel/common"
	"github.com/lunixbochs/usercorn/kernel/linux"
)

// MinusOne represents -1 when interpreted as signed integer.
// This is often used to indicate an error.
const MinusOne = 0xFFFFFFFFFFFFFFFF

// VirtualLinuxKernel is a kernel that isolates processes from the host.
type VirtualLinuxKernel struct {
	*co.KernelBase
	Unpack func(co.Buf, interface{})
	Fs     *ramfs.Filesystem
	Fds    map[co.Fd]File // Open file descriptors
	nextfd co.Fd
}

type netFile struct {
	net.Conn
}

func (f *netFile) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("stat on netfile not implemented")
}

func (f *netFile) Truncate(int64) error {
	return fmt.Errorf("truncate on netfile not implemented")
}

// NewVirtualKernel creates a Linux Kernel that is isolated from the operating system.
func NewVirtualKernel() *VirtualLinuxKernel {
	kernel := &VirtualLinuxKernel{
		KernelBase: &co.KernelBase{},
		Fs:         ramfs.New(),
		Fds:        map[co.Fd]File{},
	}
	kernel.Argjoy.Register(func(arg interface{}, vals []interface{}) error {
		return linux.Unpack(kernel, arg, vals)
	})
	kernel.initFs()
	return kernel
}

func (k *VirtualLinuxKernel) initFs() {
	// Stdout
	k.Fds[0] = os.Stdin
	k.Fds[1] = os.Stdout
	k.nextfd = 3
}

// StdinOutPort redirects stdin and stout to the connection that connects
// to the specified port.
func (k *VirtualLinuxKernel) StdinOutPort(port int) error {
	addr := fmt.Sprintf("localhost:%d", port)
	log.Printf("Listen on %q for incoming connection", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()
	c, err := l.Accept()
	if err != nil {
		return err
	}
	nf := &netFile{c}
	k.Fds[0] = nf
	k.Fds[1] = nf
	return nil
}
