package common

import (
	"bytes"
	"io"

	"github.com/lunixbochs/argjoy"
	"github.com/pkg/errors"
)

func (k *KernelBase) readCStringAt(addr uint64, maxlen int) (string, error) {
	m := k.U.Mem()
	m.Seek(int64(addr), io.SeekStart)
	var buff bytes.Buffer
	t := []byte{0}
	for i := 0; i < maxlen; i++ {
		_, err := m.Read(t)
		if err != nil {
			return buff.String(), err
		}
		if t[0] == 0 {
			return buff.String(), nil
		}
		buff.Write(t)
	}
	return buff.String(), nil
}

func (k *KernelBase) commonArgCodec(arg interface{}, vals []interface{}) error {
	if reg, ok := vals[0].(uint64); ok {
		switch v := arg.(type) {
		case *Buf:
			*v = NewBuf(k, reg)
		case *Obuf:
			*v = Obuf{NewBuf(k, reg)}
		case *Len:
			*v = Len(reg)
		case *Off:
			*v = Off(reg)
		case *Fd:
			*v = Fd(reg)
		case *Ptr:
			*v = Ptr(reg)
		case *string:
			s, err := k.readCStringAt(reg, 255)
			if err != nil {
				return errors.Wrapf(err, "ReadStrAt(%#x) failed", reg)
			}
			*v = s
		default:
			return argjoy.NoMatch
		}
		return nil
	}
	return argjoy.NoMatch
}
