package arch

import (
	"github.com/pkg/errors"

	"github.com/felberj/binemu/arch/arm"
	"github.com/felberj/binemu/arch/arm64"
	"github.com/felberj/binemu/arch/m68k"
	"github.com/felberj/binemu/arch/mips"
	"github.com/felberj/binemu/arch/sparc"
	"github.com/felberj/binemu/arch/x86"
	"github.com/felberj/binemu/arch/x86_16"
	"github.com/felberj/binemu/arch/x86_64"
	"github.com/felberj/binemu/models"
)

var archMap = map[string]*models.Arch{
	"arm":    arm.Arch,
	"arm64":  arm64.Arch,
	"m68k":   m68k.Arch,
	"mips":   mips.Arch,
	"sparc":  sparc.Arch,
	"x86":    x86.Arch,
	"x86_16": x86_16.Arch,
	"x86_64": x86_64.Arch,
}

func GetArch(name, os string) (*models.Arch, *models.OS, error) {
	a, ok := archMap[name]
	if !ok {
		return nil, nil, errors.Errorf("Arch '%s' not found.", name)
	}
	o, ok := a.OS[os]
	if !ok {
		return nil, nil, errors.Errorf("OS '%s' not found for arch '%s'.", os, name)
	}
	return a, o, nil
}
