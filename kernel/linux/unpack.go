package linux

import (
	"syscall"

	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/kernel/linux/unpack"
	"github.com/felberj/binemu/native"
	"github.com/felberj/binemu/native/enum"
	"github.com/lunixbochs/argjoy"
)

func Unpack(k co.Kernel, arg interface{}, vals []interface{}) error {
	reg0 := vals[0].(uint64)
	// null pointer guard
	if reg0 == 0 {
		// work around syscall package panicking on null Sockaddr
		switch v := arg.(type) {
		case *syscall.Sockaddr:
			*v = &syscall.SockaddrInet4{}
		}
		return nil
	}
	buf := co.NewBuf(k, reg0)
	switch v := arg.(type) {
	case *syscall.Sockaddr:
		*v = unpack.Sockaddr(buf, int(vals[1].(uint64)))
	case **syscall.Timeval:
		tmp := &native.Timeval{}
		if err := buf.Unpack(tmp); err != nil {
			return err
		}
		nsec := tmp.Sec*1e9 + tmp.Usec*1e3
		*v = &syscall.Timeval{}
		**v = syscall.NsecToTimeval(nsec)
	case **native.Fdset32:
		tmp := &native.Fdset32{}
		if err := buf.Unpack(tmp); err != nil {
			return err
		}
		*v = tmp
	case **native.Timespec:
		tmp := &native.Timespec{}
		if err := buf.Unpack(tmp); err != nil {
			return err
		}
		*v = tmp
	case *enum.OpenFlag:
		*v = unpack.OpenFlag(reg0)
	case *enum.MmapFlag:
		*v = unpack.MmapFlag(reg0)
	case *enum.MmapProt:
		*v = unpack.MmapProt(reg0)
	default:
		return argjoy.NoMatch
	}
	return nil
}

func registerUnpack(k *LinuxKernel) {
	k.Argjoy.Register(func(arg interface{}, vals []interface{}) error {
		return Unpack(k, arg, vals)
	})
}
