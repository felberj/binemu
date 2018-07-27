package cpu

import (
	"bytes"
	"fmt"
	"strings"
)

type FileDesc struct {
	Name string
	Off  uint64
	Len  uint64
}

func (f *FileDesc) shift(off uint64) *FileDesc {
	if f != nil && off < f.Len {
		return &FileDesc{
			Name: f.Name,
			Len:  f.Len - off,
			Off:  f.Off + off,
		}
	}
	return f
}

type Page struct {
	Addr uint64
	Size uint64
	Prot int
	Data []byte

	Desc string
	File *FileDesc
}

func (p *Page) String() string {
	// add prot
	prots := []int{PROT_READ, PROT_WRITE, PROT_EXEC}
	chars := []string{"r", "w", "x"}
	prot := ""
	for i := range prots {
		if p.Prot&prots[i] != 0 {
			prot += chars[i]
		} else {
			prot += "-"
		}
	}
	desc := fmt.Sprintf("0x%x-0x%x %s", p.Addr, p.Addr+p.Size, prot)
	if p.Desc != "" {
		desc += fmt.Sprintf(" [%s]", p.Desc)
	}
	if p.File != nil {
		desc += fmt.Sprintf(" %s", p.File.Name)
		if p.File.Off > 0 {
			desc += fmt.Sprintf("(%#x)", p.File.Off)
		}
	}
	return desc
}

func (p *Page) Contains(addr uint64) bool {
	return addr >= p.Addr && addr < p.Addr+p.Size
}

// start = max(s1, s2), end = min(e1, e2), ok = end > start
func (p *Page) Intersect(addr, size uint64) (uint64, uint64, bool) {
	start := p.Addr
	end := p.Addr + p.Size
	e2 := addr + size
	if end > e2 {
		end = e2
	}
	if start < addr {
		start = addr
	}
	return start, end - start, end > start
}

func (p *Page) Overlaps(addr, size uint64) bool {
	_, _, ok := p.Intersect(addr, size)
	return ok
}

/*
// how to slice a page
off1         len1
addr1        |  size1
|            |  |
[     page      ]
[               ]
[  [ slice ]    ]
   |       |
   addr2   size2
   off2    len2

delta = addr2 - addr1
if delta < len1 {
    len2 = len1 - delta
    off2 = off1 + delta
}
*/
func (p *Page) Slice(addr, size uint64) *Page {
	o := addr - p.Addr
	file := p.File.shift(o)
	data := p.Data
	if p.Data != nil {
		data = p.Data[o : o+size]
	}
	return &Page{Addr: addr, Size: size, Prot: p.Prot, Data: data, Desc: p.Desc, File: file}
}

/*
// how to split a page, simple edition //
laddr                      rsize
|      lsize       raddr   |
[------|----page---|-------]
[-left-][---mid---][-right-]
|       |         |        |
|       addr      size     |
paddr                      psize

laddr = paddr
lsize = addr - paddr

raddr = addr + size
rsize = (paddr + psize) - raddr

// how to split a page, overlap edition //
addr                 size
|                    |
[--------mid---------]
      [--page--]
      |        |
      paddr    psize

pad(addr, paddr - addr, 0)
pend = paddr + psize
pad(pend, size - pend, 0)
*/
func (p *Page) Split(addr, size uint64) (left, right *Page) {
	// build page for raddr:rsize
	if addr+size < p.Addr+p.Size {
		ra := addr + size
		rs := (p.Addr + p.Size) - ra
		right = p.Slice(ra, rs)
		if p.Data != nil {
			p.Data = p.Data[:ra-p.Addr]
		}
	}
	// space on the left
	if addr > p.Addr {
		ls := addr - p.Addr
		left = p.Slice(p.Addr, ls)
		if p.Data != nil {
			p.Data = p.Data[ls:]
		}
	}
	// pad the middle
	if addr < p.Addr {
		extra := bytes.Repeat([]byte{0}, int(p.Addr-addr))
		if p.Data != nil {
			p.Data = append(extra, p.Data...)
		}
	}
	// pad the end
	raddr, nraddr := p.Addr+p.Size, addr+size
	if nraddr > raddr {
		extra := bytes.Repeat([]byte{0}, int(nraddr-raddr))
		if p.Data != nil {
			p.Data = append(p.Data, extra...)
		}
	}
	p.File = p.File.shift(addr - p.Addr)
	p.Addr, p.Size = addr, size
	return left, right
}

func (pg *Page) Write(addr uint64, p []byte) {
	copy(pg.Data[addr-pg.Addr:], p)
}

type Pages []*Page

func (p Pages) Len() int           { return len(p) }
func (p Pages) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Pages) Less(i, j int) bool { return p[i].Addr < p[j].Addr }

func (p Pages) String() string {
	s := make([]string, len(p))
	for i, v := range p {
		s[i] = v.String()
	}
	return strings.Join(s, "\n")
}

// binary search to find index of first region containing addr, if any, else -1
func (p Pages) bsearch(addr uint64) (int, int) {
	l := 0
	r := len(p) - 1
	for l <= r {
		mid := (l + r) / 2
		e := p[mid]
		if addr >= e.Addr {
			if addr < e.Addr+e.Size {
				return mid, mid
			}
			l = mid + 1
		} else if addr < e.Addr {
			r = mid - 1
		}
	}
	return l, -1
}

func (p Pages) Find(addr uint64) *Page {
	_, i := p.bsearch(addr)
	if i >= 0 {
		return p[i]
	}
	return nil
}

func (p Pages) FindRange(start, size uint64) []*Page {
	pos, _ := p.bsearch(start)
	st := pos
	for pos >= 0 && pos < len(p) && p[pos].Overlaps(start, size) {
		pos++
	}
	return p[st:pos]
}
