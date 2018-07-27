package linux

const (
	DT_UNKNOWN = 0
	DT_FIFO    = 1
	DT_CHR     = 2
	DT_DIR     = 4
	DT_BLK     = 6
	DT_REG     = 8
	DT_LNK     = 10
	DT_SOCK    = 12
	DT_WHT     = 14
)

// TODO: need to differentiate between guests with and without LFS
type Dirent struct {
	Ino  uint64 `struc:"uint64"`
	Off  uint64 `struc:"uint64"`
	Len  int    `struc:"uint16"`
	Name string
	Type int `struc:"uint8"`
}

type Dirent64 struct {
	Ino  uint64 `struc:"uint64"`
	Off  uint64 `struc:"uint64"`
	Len  int    `struc:"uint16"`
	Type int    `struc:"uint8"`
	Name string
}
