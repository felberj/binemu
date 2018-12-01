package main

import (
	"github.com/felberj/binemu/cmd"

	_ "github.com/felberj/binemu/cmd/run"

	_ "github.com/felberj/binemu/cmd/cfg"
	_ "github.com/felberj/binemu/cmd/cgc"
	_ "github.com/felberj/binemu/cmd/com"
	_ "github.com/felberj/binemu/cmd/fuzz"
	_ "github.com/felberj/binemu/cmd/imgtrace"
	_ "github.com/felberj/binemu/cmd/repl"
	_ "github.com/felberj/binemu/cmd/shellcode"
	_ "github.com/felberj/binemu/cmd/trace"
)

func main() { cmd.Main() }
