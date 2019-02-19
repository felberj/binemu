package vm

import (
	"github.com/felberj/binemu/models"
	"github.com/felberj/binemu/models/cpu"
)

// Process represents a process within a virtual machine
type Process struct {
	ID          int
	Executable  string
	Args        []string
	Environment []string

	vm   *VM
	cpu  cpu.Cpu
	os   *models.OS
	arch *models.Arch
}

func (p *Process) initHooks() {
	//	invalid := cpu.HOOK_MEM_ERR

}
