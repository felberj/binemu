package vm

import (
	"io"
	"os"
	"path"

	"github.com/felberj/binemu/arch"
	"github.com/felberj/binemu/loader"
	"github.com/felberj/ramfs"
	"github.com/pkg/errors"

	pb "github.com/felberj/binemu/proto_gen"
)

// VM is the environment the binary should be emulated in.
type VM struct {
	Fs         *ramfs.Filesystem
	currentPid int
}

// LoadFiles loads the files from the config into the filesytem of the environment.
func (v *VM) LoadFiles(c *pb.Config) error {
	for _, p := range c.Files {
		f := path.Join(c.ConfigDir, p.HostPath)
		if err := v.Fs.MapFile(f, p.GuestPath); err != nil {
			return errors.Wrapf(err, "unable to load file %q into ramfs to %q", f, p.GuestPath)
		}
		if err := v.Fs.Chmod(p.GuestPath, os.FileMode(p.Mode)); err != nil {
			return errors.Wrapf(err, "unable to chmod %q", p.GuestPath)
		}
	}
	return nil
}

// Process creates a new process for the provided executable.
// - create unicorn instance
// - prepare memory and kernel for the process
// - load loader into virtual machine (if required)
// If a loader is required, the binary is written to the filesystem and the entry point
// it set to the loader.
func (v *VM) Process(c *pb.Config, exec string, args, envornment []string) (*Process, error) {
	exe, err := os.Open(exec)
	if err != nil {
		return nil, err
	}
	defer exe.Close()
	l, err := loader.Loader(exe)
	if err != nil {
		return nil, err
	}
	if l.Interp() != "" {
		if c.Loader == "" {
			return nil, errors.New("the binary is interp, but no loader in config")
		}
		// copy the binary into the virual machine
		vpath := path.Join("/bin", path.Base(exec))
		vexe, err := v.Fs.Create(vpath)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to open virtual file")
		}
		defer vexe.Close()
		exe.Seek(0, 0)
		if _, err := io.Copy(vexe, exe); err != nil {
			return nil, err
		}
		ldargs := []string{"ld", vpath}
		ldargs = append(ldargs, args...)
		args = ldargs
		exec = path.Join(c.ConfigDir, c.Loader)
		exe, err = os.Open(exec)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to open loader")
		}
		defer exe.Close()
		l, err = loader.Loader(exe)
		if err != nil {
			return nil, err
		}
	}
	a, os, err := arch.GetArch(l.Arch(), c.Kernel)
	if err != nil {
		return nil, err
	}
	cpu, err := a.Cpu.New()
	if err != nil {
		return nil, err
	}

	/* task := NewTask(cpu, a, os, l.ByteOrder())
	u := NewUsercornWrapper(exe, task, fs, l, os, &models.Config{
		Args: args,
	})
	if err := u.LoadBinary(f); err != nil {
		return nil, err
	}*/
	v.currentPid++
	return &Process{
		ID:          v.currentPid,
		Executable:  exec,
		Args:        args,
		Environment: envornment,
		cpu:         cpu,
		vm:          v,
		os:          os,
		arch:        a,
	}, nil

}

// NewVM creates a new virtual environment to run binaries in.
func NewVM() *VM {
	return &VM{
		Fs: ramfs.New(),
	}
}
