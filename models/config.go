package models

// Config allows to configure binemu
type Config struct {
	Env  []string
	Args []string

	Strsize int
	Verbose bool

	BlockSyscalls bool
}
