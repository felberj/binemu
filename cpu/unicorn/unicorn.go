package unicorn

import (
	"github.com/pkg/errors"

	uc "github.com/felberj/binemu/unicorn"
)

type Builder struct {
	Arch, Mode int
}

func (b *Builder) New() (*Cpu, error) {
	u, err := uc.NewUnicorn(b.Arch, b.Mode)
	if err != nil {
		return nil, errors.Wrap(err, "NewUnicorn() failed")
	}
	return &Cpu{u}, nil
}

type Cpu struct {
	*uc.Unicorn
}

func (u *Cpu) Backend() interface{} {
	return u.Unicorn
}

func (u *Cpu) ContextSave(reuse interface{}) (interface{}, error) {
	if reuse == nil {
		return u.Unicorn.ContextSave(nil)
	}
	return u.Unicorn.ContextSave(reuse.(uc.Context))
}

func (u *Cpu) ContextRestore(ctx interface{}) error {
	return u.Unicorn.ContextRestore(ctx.(uc.Context))
}

func (u *Cpu) MemMap(addr, size uint64, prot int) error {
	return u.Unicorn.MemMapProt(addr, size, prot)
}

func (u *Cpu) MemProt(addr, size uint64, prot int) error {
	return u.Unicorn.MemProtect(addr, size, prot)
}

func (u *Cpu) MemRegions() ([]*uc.MemRegion, error) {
	return u.Unicorn.MemRegions()
}
