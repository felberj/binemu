package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	usercorn "github.com/felberj/binemu"
	"github.com/felberj/binemu/arch"
	"github.com/felberj/binemu/loader"
	"github.com/felberj/binemu/models"
	pb "github.com/felberj/binemu/proto_gen"
	"github.com/felberj/ramfs"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	config     = flag.String("config", "", "configuration of the environment (as textproto)")
	configPath = flag.String("config_path", "", "path to the configurations file")
)

func loadFiles(fs *ramfs.Filesystem, c *pb.Config) error {
	for _, p := range c.Files {
		f := path.Join(c.ConfigDir, p.HostPath)
		if err := fs.MapFile(f, p.GuestPath); err != nil {
			return errors.Wrapf(err, "unable to load file %q into ramfs to %q", f, p.GuestPath)
		}
		if err := fs.Chmod(p.GuestPath, os.FileMode(p.Mode)); err != nil {
			return errors.Wrapf(err, "unable to chmod %q", p.GuestPath)
		}
	}
	return nil
}

func run(c *pb.Config, args []string) error {
	fs := ramfs.New()

	exe := args[0]
	f, err := os.Open(exe)
	if err != nil {
		return err
	}
	defer f.Close()
	l, err := loader.Loader(f)
	if err != nil {
		return err
	}
	if l.Interp() != "" {
		if c.Loader == "" {
			return errors.New("the binary is interp, but no loader in config")
		}
		a := []string{"ld", "/bin/exec"}
		a = append(a, args[1:]...)
		args = a
		e, err := fs.Create("/bin/exec")
		if err != nil {
			return nil
		}
		defer e.Close()
		if _, err := io.Copy(e, f); err != nil {
			return err
		}
		exe = path.Join(c.ConfigDir, c.Loader)
		f, err = os.Open(exe)
		if err != nil {
			return errors.Wrapf(err, "unable to open loader")
		}
		defer f.Close()
		l, err = loader.Loader(f)
		if err != nil {
			return err
		}
	}
	a, os, err := arch.GetArch(l.Arch(), c.Kernel)
	if err != nil {
		return err
	}
	cpu, err := a.Cpu.New()
	if err != nil {
		return err
	}
	if err := loadFiles(fs, c); err != nil {
		return err
	}
	task := usercorn.NewTask(cpu, a, os, l.ByteOrder())
	u := usercorn.NewUsercornWrapper(exe, task, fs, l, os, &models.Config{
		Args: args,
	})
	if err := u.LoadBinary(f); err != nil {
		return err
	}
	return u.Run()
}

func main() {
	flag.Parse()
	var c pb.Config
	if *configPath != "" {
		d, err := ioutil.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("unable to read config file: %v", err)
		}
		if err := proto.UnmarshalText(string(d), &c); err != nil {
			log.Fatalf("Unable to parse config: %v", err)
		}
		c.ConfigDir = path.Dir(*configPath)
	} else {
		if err := proto.UnmarshalText(*config, &c); err != nil {
			log.Fatalf("Unable to parse config: %v", err)
		}
		c.ConfigDir = "./"
	}
	args := flag.Args()
	if len(args) == 0 {
		log.Fatalf("No program specified")
	}
	if err := run(&c, args); err != nil {
		log.Fatalf("Error while running the binary: %v", err)
	}
}
