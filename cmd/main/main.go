package main

import (
	"github.com/lunixbochs/usercorn/cmd"

	_ "github.com/lunixbochs/usercorn/cmd/run"

	_ "github.com/lunixbochs/usercorn/cmd/cfg"
	_ "github.com/lunixbochs/usercorn/cmd/cgc"
	_ "github.com/lunixbochs/usercorn/cmd/com"
	_ "github.com/lunixbochs/usercorn/cmd/fuzz"
	_ "github.com/lunixbochs/usercorn/cmd/imgtrace"
	_ "github.com/lunixbochs/usercorn/cmd/repl"
	_ "github.com/lunixbochs/usercorn/cmd/shellcode"
	_ "github.com/lunixbochs/usercorn/cmd/trace"
)

func main() { cmd.Main() }
