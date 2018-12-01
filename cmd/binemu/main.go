package main

import (
	"flag"
	"log"
	"os"

	usercorn "github.com/felberj/binemu"
	"github.com/felberj/binemu/arch"
	"github.com/felberj/binemu/loader"
	"github.com/felberj/binemu/models"
)

var (
	executable = flag.String("executable", "", "the binary that should be emulated")
)

func run(executable, kernel string) error {
	f, err := os.Open(executable)
	if err != nil {
		return err
	}
	defer f.Close()
	l, err := loader.Loader(f)
	if err != nil {
		return err
	}
	a, os, err := arch.GetArch(l.Arch(), kernel)
	if err != nil {
		return err
	}
	cpu, err := a.Cpu.New()
	if err != nil {
		return err
	}
	task := usercorn.NewTask(cpu, a, os, l.ByteOrder())
	u := usercorn.NewUsercornWrapper(executable, task, l, os, &models.Config{
		Args: []string{executable},
	})
	if err := u.LoadBinary(f); err != nil {
		return err
	}
	return u.Run()
}

func run2(executable string) error {
	u, err := usercorn.NewUsercorn(executable, &models.Config{})
	if err != nil {
		return err
	}
	return u.Run()
}

func main() {
	flag.Parse()
	if *executable == "" {
		flag.Usage()
		log.Fatal("No executable provided")
	}
	if err := run(*executable, "virtual-linux"); err != nil {
		log.Fatalf("Error while running the binary: %v", err)
	}
}
