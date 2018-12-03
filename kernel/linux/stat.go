package linux

// TODO: use FileInfo instead of nonportable Syscall interface?
import (
	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/models"
)

func NewLinuxStat_generic(stat *LinuxStat64_x86, bits uint, large bool) interface{} {
	return &LinuxStat_generic{
		Dev:       uint32(stat.Dev),
		Ino:       uint64(stat.Ino),
		Mode:      uint32(stat.Mode),
		Uid:       stat.Uid,
		Gid:       stat.Gid,
		Rdev:      uint32(stat.Rdev),
		Size:      int64(stat.Size),
		Blksize:   uint32(stat.Blksize),
		Blkcnt:    uint64(stat.Blkcnt),
		Atime:     uint32(stat.Atime),
		AtimeNsec: uint32(stat.AtimeNsec),
		Mtime:     uint32(stat.Mtime),
		MtimeNsec: uint32(stat.MtimeNsec),
		Ctime:     uint32(stat.Ctime),
		CtimeNsec: uint32(stat.CtimeNsec),
	}
}

func NewLinuxStat_x86(stat *LinuxStat64_x86, bits uint, large bool) interface{} {
	if bits == 64 {
		return &LinuxStat64_x86{
			Dev:       uint64(stat.Dev),
			Ino:       uint64(stat.Ino),
			Mode:      uint32(stat.Mode),
			Uid:       stat.Uid,
			Gid:       stat.Gid,
			Rdev:      uint64(stat.Rdev),
			Size:      int64(stat.Size),
			Blksize:   int64(stat.Blksize),
			Blkcnt:    int64(stat.Blkcnt),
			Atime:     uint64(stat.Atime),
			AtimeNsec: uint64(stat.AtimeNsec),
			Mtime:     uint64(stat.Mtime),
			MtimeNsec: uint64(stat.MtimeNsec),
			Ctime:     uint64(stat.Ctime),
			CtimeNsec: uint64(stat.CtimeNsec),
		}
	} else {
		if large {
			return &Linux32Stat64_x86{
				Dev:       uint64(stat.Dev),
				Ino:       uint32(stat.Ino),
				Mode:      uint32(stat.Mode),
				Uid:       stat.Uid,
				Gid:       stat.Gid,
				Rdev:      uint64(stat.Rdev),
				Size:      int64(stat.Size),
				Blksize:   uint32(stat.Blksize),
				Blkcnt:    uint64(stat.Blkcnt),
				Atime:     uint32(stat.Atime),
				AtimeNsec: uint32(stat.AtimeNsec),
				Mtime:     uint32(stat.Mtime),
				MtimeNsec: uint32(stat.MtimeNsec),
				Ctime:     uint32(stat.Ctime),
				CtimeNsec: uint32(stat.CtimeNsec),
				LongIno:   uint64(stat.Ino),
			}
		}
		return &Linux32Stat_x86{
			Dev:       uint32(stat.Dev),
			Ino:       uint32(stat.Ino),
			Mode:      uint16(stat.Mode),
			Uid:       stat.Uid,
			Gid:       stat.Gid,
			Rdev:      uint32(stat.Rdev),
			Size:      uint32(stat.Size),
			Blksize:   uint32(stat.Blksize),
			Blkcnt:    uint32(stat.Blkcnt),
			Atime:     uint32(stat.Atime),
			AtimeNsec: uint32(stat.AtimeNsec),
			Mtime:     uint32(stat.Mtime),
			MtimeNsec: uint32(stat.MtimeNsec),
			Ctime:     uint32(stat.Ctime),
			CtimeNsec: uint32(stat.CtimeNsec),
		}
	}
}

func NewDarwinStat(stat *LinuxStat64_x86, bits uint, large bool) interface{} {
	if bits == 64 {
		return &DarwinStat64{
			Dev:       int32(stat.Dev),
			Mode:      uint16(stat.Mode),
			Ino:       uint64(stat.Ino),
			Uid:       uint32(stat.Uid),
			Gid:       uint32(stat.Gid),
			Rdev:      int32(stat.Rdev),
			Size:      int64(stat.Size),
			Blksize:   int32(stat.Blksize),
			Blkcnt:    int64(stat.Blkcnt),
			Atime:     int64(stat.Atime),
			AtimeNsec: int64(stat.AtimeNsec),
			Mtime:     int64(stat.Mtime),
			MtimeNsec: int64(stat.MtimeNsec),
			Ctime:     int64(stat.Ctime),
			CtimeNsec: int64(stat.CtimeNsec),
		}
	} else {
		panic("darwin-32 stat struct unimplemented")
	}
}

func HandleStat(buf co.Obuf, stat *LinuxStat64_x86, u models.Usercorn, large bool) uint64 {
	var pack interface{}
	os, bits := u.OS(), u.Bits()
	arch := u.Arch().Name
	switch os {
	case "virtual-linux":
		fallthrough
	case "linux":
		switch arch {
		case "x86":
			fallthrough
		case "x86_64":
			pack = NewLinuxStat_x86(stat, bits, large)
		default:
			pack = NewLinuxStat_generic(stat, bits, large)
		}
	case "darwin":
		pack = NewDarwinStat(stat, bits, large)
	default:
		panic("(currently) unsupported target OS for fstat: " + os)
	}
	if err := buf.Pack(pack); err != nil {
		panic(err)
	}
	return 0
}

type LinuxStat_generic struct {
	Dev uint32
	Ino uint64

	Mode     uint32
	Nlink    uint32
	Uid, Gid uint32
	Rdev     uint32
	Pad0     [4]byte

	Size    int64
	Blksize uint32
	Pad1    [4]byte
	Blkcnt  uint64

	Atime     uint32
	AtimeNsec uint32
	Mtime     uint32
	MtimeNsec uint32
	Ctime     uint32
	CtimeNsec uint32

	Reserved [8]byte
}

type Linux32Stat_x86 struct {
	Dev      uint32
	Ino      uint32
	Mode     uint16
	Nlink    uint16
	Uid, Gid uint32
	Rdev     uint32
	Size     uint32
	Blksize  uint32
	Blkcnt   uint32

	Atime     uint32
	AtimeNsec uint32
	Mtime     uint32
	MtimeNsec uint32
	Ctime     uint32
	CtimeNsec uint32

	Reserved4 uint32
	Reserved5 uint32
}

type Linux32Stat64_x86 struct {
	Dev  uint64
	Pad0 [4]byte
	Ino  uint32

	Mode     uint32
	Nlink    uint32
	Uid, Gid uint32
	Rdev     uint64
	Pad3     [4]byte

	Size    int64
	Blksize uint32
	Blkcnt  uint64

	Atime     uint32
	AtimeNsec uint32
	Mtime     uint32
	MtimeNsec uint32
	Ctime     uint32
	CtimeNsec uint32

	LongIno uint64
}

type LinuxStat64_x86 struct {
	Dev      uint64
	Ino      uint64
	Nlink    uint64
	Mode     uint32
	Uid, Gid uint32
	Pad0     uint32
	Rdev     uint64
	Size     int64
	Blksize  int64
	Blkcnt   int64

	Atime     uint64
	AtimeNsec uint64
	Mtime     uint64
	MtimeNsec uint64
	Ctime     uint64
	CtimeNsec uint64

	Reserved3 [3]uint64
}

type DarwinStat struct {
}

type DarwinStat64 struct {
	Dev   int32  /* [XSI] ID of device containing file */
	Mode  uint16 /* [XSI] Mode of file (see below) */
	Nlink uint16 /* [XSI] Number of hard links */
	Ino   uint64 /* [XSI] File serial number */
	Uid   uint32 /* [XSI] User ID of the file */
	Gid   uint32 /* [XSI] Group ID of the file */
	Rdev  int32  /* [XSI] Device ID */
	Pad0  int32  /* padding to align the embedded structs */

	Atime         int64 /* time of last access */
	AtimeNsec     int64 /* time of last access */
	Mtime         int64 /* time of last data modification */
	MtimeNsec     int64 /* time of last data modification */
	Ctime         int64 /* time of last status change */
	CtimeNsec     int64 /* time of last status change */
	Birthtime     int64 /* time of file creation(birth) */
	BirthtimeNsec int64 /* time of file creation(birth) */

	Size      int64    /* [XSI] file size, in bytes */
	Blkcnt    int64    /* [XSI] blocks allocated for file */
	Blksize   int32    /* [XSI] optimal blocksize for I/O */
	st_flags  uint32   /* user defined flags for file */
	st_gen    uint32   /* file generation number */
	st_lspare int32    /* RESERVED: DO NOT USE! */
	st_qspare [2]int64 /* RESERVED: DO NOT USE! */
}
