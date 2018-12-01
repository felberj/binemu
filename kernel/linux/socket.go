package linux

import (
	"syscall"

	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/native"
)

func fdcount(bufs ...co.Obuf) int {
	count := 0
	for _, b := range bufs {
		if b.Addr != 0 {
			var f native.Fdset32
			b.Unpack(&f)
			count += len(f.Fds())
		}
	}
	return count
}

func (k *LinuxKernel) Select(args []co.Obuf, nfds int, readfds, writefds, errorfds *native.Fdset32, timeout *syscall.Timeval) uint64 {
	if errno := k.PosixKernel.Select(args, nfds, readfds, writefds, errorfds, timeout); errno != 0 {
		return errno
	}
	return uint64(fdcount(args[1], args[2], args[3]))
}