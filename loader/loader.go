package loader

import (
	"debug/dwarf"
	"encoding/binary"
)

// NoOSHint indicates that there is no os hint
const NoOSHint = ""

type SegmentData struct {
	Off        uint64
	Addr, Size uint64
	Prot       int
	DataFunc   func() ([]byte, error)
}

func (s *SegmentData) Data() ([]byte, error) {
	return s.DataFunc()
}

func (s *SegmentData) ContainsPhys(addr uint64) bool {
	return s.Off <= addr && addr < s.Off+s.Size
}

func (s *SegmentData) ContainsVirt(addr uint64) bool {
	return s.Addr <= addr && addr < s.Addr+s.Size
}

type Segment struct {
	Start, End uint64
	Prot       int
}

func (s *Segment) Overlaps(o *Segment) bool {
	return (s.Start >= o.Start && s.Start < o.End) || (o.Start >= s.Start && o.Start < s.End)
}

func (s *Segment) Merge(o *Segment) {
	if s.Start > o.Start {
		s.Start = o.Start
	}
	if s.End < o.End {
		s.End = o.End
	}
}

type Loader interface {
	Arch() string
	Bits() int
	ByteOrder() binary.ByteOrder
	OS() string
	Entry() uint64
	Type() int
	Interp() string
	Header() (uint64, []byte, int)
	Symbols() ([]Symbol, error)
	Segments() ([]SegmentData, error)
	DataSegment() (uint64, uint64)
	DWARF() (*dwarf.Data, error)
}

type LoaderBase struct {
	arch      string
	bits      int
	byteOrder binary.ByteOrder
	os        string
	entry     uint64
	symCache  []Symbol
}

func (l *LoaderBase) Arch() string {
	return l.arch
}

func (l *LoaderBase) Bits() int {
	return l.bits
}

func (l *LoaderBase) ByteOrder() binary.ByteOrder {
	if l.byteOrder == nil {
		return binary.LittleEndian
	}
	return l.byteOrder
}

func (l *LoaderBase) OS() string {
	return l.os
}

func (l *LoaderBase) Entry() uint64 {
	return l.entry
}

// Everything below this line is a stub, intended to be reimplemented by the struct embedding LoaderBase.
// These methods are defined to allow implementing a partial loader.

func (l *LoaderBase) DWARF() (*dwarf.Data, error) {
	return nil, nil
}

func (l *LoaderBase) DataSegment() (uint64, uint64) {
	return 0, 0
}

func (l *LoaderBase) Header() (uint64, []byte, int) {
	return 0, nil, 0
}

func (l *LoaderBase) Interp() string {
	return ""
}

func (l *LoaderBase) Segments() ([]SegmentData, error) {
	return nil, nil
}

func (l *LoaderBase) Symbols() ([]Symbol, error) {
	return nil, nil
}

func (l *LoaderBase) Type() int {
	return EXEC
}
