package linux

import (
	co "github.com/felberj/binemu/kernel/common"
)

const (
	FUTEX_WAIT            = 0
	FUTEX_WAKE            = 1
	FUTEX_FD              = 2
	FUTEX_REQUEUE         = 3
	FUTEX_CMP_REQUEUE     = 4
	FUTEX_WAKE_OP         = 5
	FUTEX_LOCK_PI         = 6
	FUTEX_UNLOCK_PI       = 7
	FUTEX_TRYLOCK_PI      = 8
	FUTEX_WAIT_BITSET     = 9
	FUTEX_WAKE_BITSET     = 10
	FUTEX_WAIT_REQUEUE_PI = 11
	FUTEX_CMP_REQUEUE_PI  = 12
)

const (
	FUTEX_PRIVATE_FLAG   = 128
	FUTEX_CLOCK_REALTIME = 256
	FUTEX_CMD_MASK       = ^(FUTEX_PRIVATE_FLAG | FUTEX_CLOCK_REALTIME)
)

var ENOSYS = 33

// SetTidAddress syscall (not implemented)
func (k *LinuxKernel) SetTidAddress(tidptr co.Buf) uint64 {
	return 0
}

// SetRobustList syscall (not implemented)
func (k *LinuxKernel) SetRobustList(tid int, head co.Buf) {}

// Futex syscall
// Timeout is a co.Buf here because some forms of futex don't pass it
func (k *LinuxKernel) Futex(uaddr co.Buf, op, val int, timeout, uaddr2 co.Buf, val3 uint64) int {
	if op&FUTEX_CLOCK_REALTIME != 0 {
		return -ENOSYS
	}
	switch op & FUTEX_CMD_MASK {
	case FUTEX_WAIT:
	case FUTEX_WAKE:
	case FUTEX_WAIT_BITSET:
	case FUTEX_WAKE_BITSET:
	default:
		return -1
	}
	return 0
}
