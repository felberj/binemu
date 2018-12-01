package trace

import (
	"encoding/binary"
	"io"
	"strings"

	"github.com/felberj/binemu/arch"
	"github.com/felberj/binemu/models"
	"github.com/lunixbochs/struc"
	"github.com/pkg/errors"
)

var TRACE_MAGIC = "UCIR"

type TraceHeader struct {
	// MAGIC ("UCIR")
	Magic string `struc:"[4]byte" json:"-"`
	// file format version
	Version uint32 `json:"version"`

	// Emulated architecture. Possible values include "x86_64", "x86", "mips", "arm", "arm64". Right-null-padded.
	Arch string `struc:"[32]byte" json:"arch"`

	// Emulated OS. Possible values include "linux", "darwin", "netbsd", "cgc", "dos". Right-null-padded.
	OS string `struc:"[32]byte" json:"os"`

	// Byte Order - 0 for little, 1 for big
	CodeOrderNum  uint8            `json:"-"`
	DataOrderNum  uint8            `json:"-"`
	CodeOrderName string           `struc:"skip" json:"code_order"`
	DataOrderName string           `struc:"skip" json:"data_order"`
	CodeOrder     binary.ByteOrder `struc:"skip" json:"-"`
	DataOrder     binary.ByteOrder `struc:"skip" json:"-"`
}

type TraceWriter struct {
	w io.WriteCloser
}

func NewWriter(w io.WriteCloser, u models.Usercorn) (*TraceWriter, error) {
	order := u.ByteOrder()
	var num uint8
	var name string
	if order == binary.LittleEndian {
		num = 0
		name = "little"
	} else if order == binary.BigEndian {
		num = 1
		name = "big"
	}
	header := &TraceHeader{
		Magic:   TRACE_MAGIC,
		Version: 1,
		Arch:    u.Loader().Arch(),
		OS:      u.Loader().OS(),

		CodeOrderNum:  num,
		DataOrderNum:  num,
		CodeOrderName: name,
		DataOrderName: name,
		CodeOrder:     order,
		DataOrder:     order,
	}
	if err := struc.Pack(w, header); err != nil {
		return nil, errors.Wrap(err, "failed to pack header")
	}
	return &TraceWriter{w: w}, nil
}

// write a frame at a time
func (t *TraceWriter) Pack(frame models.Op) error {
	tmp := make([]byte, frame.Sizeof())
	frame.Pack(tmp)
	_, err := t.w.Write(tmp)
	return err
}

func (t *TraceWriter) Close() {
	t.w.Close()
}

type TraceReader struct {
	r      io.ReadCloser
	Header TraceHeader

	Arch *models.Arch
	OS   *models.OS
}

func NewReader(r io.ReadCloser) (*TraceReader, error) {
	t := &TraceReader{r: r}
	if err := struc.Unpack(r, &t.Header); err != nil {
		return nil, errors.Wrap(err, "failed to unpack header")
	}
	if t.Header.Magic != TRACE_MAGIC {
		return nil, errors.New("invalid trace file magic")
	}
	t.Header.Arch = strings.TrimRight(t.Header.Arch, "\x00")
	t.Header.OS = strings.TrimRight(t.Header.OS, "\x00")
	// look up byte order
	switch t.Header.CodeOrderNum {
	case 0:
		t.Header.CodeOrder = binary.LittleEndian
		t.Header.CodeOrderName = "little"
	case 1:
		t.Header.CodeOrder = binary.BigEndian
		t.Header.CodeOrderName = "big"
	}
	switch t.Header.DataOrderNum {
	case 0:
		t.Header.DataOrder = binary.LittleEndian
		t.Header.DataOrderName = "little"
	case 1:
		t.Header.DataOrder = binary.BigEndian
		t.Header.DataOrderName = "big"
	}
	var err error
	t.Arch, t.OS, err = arch.GetArch(t.Header.Arch, t.Header.OS)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get arch/OS")
	}
	return t, nil
}

func (t *TraceReader) Next() (models.Op, error) {
	op, _, err := Unpack(t.r, false)
	return op, err
}

func (t *TraceReader) Close() {
	t.r.Close()
}
