// Package linux provides a kernel that wants to separate the
// running process from the host system.
package linux

import (
	"fmt"
	"log"
	"net"
	"os"

	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/models"
	"github.com/felberj/ramfs"
)

const (
	STACK_BASE = 0xbf800000
	STACK_SIZE = 0x00800000
	// MinusOne represents -1 when interpreted as signed integer.
	// This is often used to indicate an error.
	MinusOne = 0xFFFFFFFFFFFFFFFF
)

// LinuxKernel is a kernel that isolates processes from the host.
type LinuxKernel struct {
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

// NewKernel creates a Linux Kernel that is isolated from the operating system.
func NewKernel(fs *ramfs.Filesystem) *LinuxKernel {
	kernel := &LinuxKernel{
		KernelBase: &co.KernelBase{},
		Fs:         fs,
		Fds:        map[co.Fd]File{},
	}
	kernel.Argjoy.Register(func(arg interface{}, vals []interface{}) error {
		return Unpack(kernel, arg, vals)
	})
	kernel.initFs()
	return kernel
}

func (k *LinuxKernel) initFs() {
	k.Fds[0] = os.Stdin
	k.Fds[1] = os.Stdout
	k.Fds[2] = os.Stderr
	k.nextfd = 3
}

// StdinOutPort redirects stdin and stout to the connection that connects
// to the specified port.
func (k *LinuxKernel) StdinOutPort(port int) error {
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

func StackInit(u models.Usercorn, args, env []string) error {
	if err := u.MapStack(STACK_BASE, STACK_SIZE, false); err != nil {
		return err
	}
	auxv, err := SetupElfAuxv(u)
	if err != nil {
		return err
	}
	if _, err := u.Push(0); err != nil {
		return err
	}
	if len(args) > 0 {
		if _, err := u.PushBytes([]byte(args[0] + "\x00")); err != nil {
			return err
		}
	}
	// push argv and envp strings
	envp, err := PushStrings(u, env...)
	if err != nil {
		return err
	}
	argv, err := PushStrings(u, args...)
	if err != nil {
		return err
	}
	// precalc envp -> argc for stack alignment
	envpb, err := PackAddrs(u, envp)
	if err != nil {
		return err
	}
	argvb, err := PackAddrs(u, argv)
	if err != nil {
		return err
	}
	var tmp [8]byte
	argcb, err := u.PackAddr(tmp[:], uint64(len(argv)))
	if err != nil {
		return err
	}
	init := append(argcb, argvb...)
	init = append(init, envpb...)
	// align stack pointer
	sp, _ := u.RegRead(u.Arch().SP)
	sp &= ^uint64(15)
	off := len(init) & 15
	if off > 0 {
		sp -= uint64(16 - off)
	}
	if err := u.RegWrite(u.Arch().SP, sp); err != nil {
		return err
	}
	// auxv
	if len(auxv) > 0 {
		if _, err := u.PushBytes(auxv); err != nil {
			return err
		}
	}
	// write envp -> argc
	_, err = u.PushBytes(init)
	return err
}

func PackAddrs(u models.Usercorn, addrs []uint64) ([]byte, error) {
	buf := make([]byte, int(u.Bits())/8*(len(addrs)+1))
	pos := buf
	for _, v := range addrs {
		x, err := u.PackAddr(pos, v)
		if err != nil {
			return nil, err
		}
		pos = pos[len(x):]
	}
	return buf, nil
}

func PushStrings(u models.Usercorn, args ...string) ([]uint64, error) {
	addrs := make([]uint64, 0, len(args)+1)
	for _, arg := range args {
		if addr, err := u.PushBytes([]byte(arg + "\x00")); err != nil {
			return nil, err
		} else {
			addrs = append(addrs, addr)
		}
	}
	return addrs, nil
}
