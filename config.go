package binemu

// ExecConfig describes the arguments and environment that should be passed to the executable.
type ExecConfig struct {
	Env  []string
	Args []string
}
