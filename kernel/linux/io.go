package linux

import (
	"crypto/md5"
	"encoding/binary"
	"io"
	"os"

	co "github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/models"
	"github.com/felberj/binemu/native/enum"
)

type File interface {
	io.ReadWriter
	io.Closer
	Stat() (os.FileInfo, error)
	Truncate(int64) error
}

// Readlink syscall
func (k *LinuxKernel) Readlink(path string, buf co.Obuf, size co.Len) uint64 {
	var name string
	if path == "/proc/self/exe" {
		name = k.U.Exe()
	} else {
		panic("Readlink not implemented")
	}
	if len(name) > int(size) {
		name = name[:size]
	}
	if err := buf.Pack([]byte(name)); err != nil {
		return MinusOne
	}
	return uint64(len(name))
}

// Access syscall
func (k *LinuxKernel) Access(path string, mode uint32) uint64 {
	f, err := k.Fs.Open(path)
	if err != nil {
		return MinusOne
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return MinusOne
	}
	if mode&1 != 0 && stat.Mode()&1 == 0 {
		return MinusOne
	}
	if mode&2 != 0 && stat.Mode()&2 == 0 {
		return MinusOne
	}
	if mode&4 != 0 && stat.Mode()&4 == 0 {
		return MinusOne
	}
	return 0
}

// Fstat syscall
func (k *LinuxKernel) Fstat(fd co.Fd, buf co.Obuf) uint64 {
	f, ok := k.Fds[fd]
	if !ok {
		return MinusOne
	}
	stat, err := f.Stat()
	if err != nil {
		return MinusOne
	}
	return handleStat(buf, stat, k.U)
}

// Write syscall
func (k *LinuxKernel) Write(fd co.Fd, buf co.Buf, size co.Len) uint64 {
	vFd, ok := k.Fds[fd]
	if !ok {
		return MinusOne
	}
	tmp := make([]byte, size)
	if err := buf.Unpack(tmp); err != nil {
		return MinusOne
	}
	n, err := vFd.Write(tmp)
	if err != nil {
		return MinusOne
	}
	return uint64(n)
}

// Writev syscall
func (k *LinuxKernel) Writev(fd co.Fd, iov co.Buf, count uint64) uint64 {
	vFd, ok := k.Fds[fd]
	if !ok {
		return MinusOne
	}
	mem := k.U.Mem()
	var written uint64
	for _, vec := range iovecIter(iov, count, k.U.Bits()) {
		if _, err := mem.Seek(int64(vec.Base), io.SeekStart); err != nil {
			return MinusOne
		}
		n, err := io.CopyN(vFd, mem, int64(vec.Len))
		if err != nil {
			return MinusOne
		}
		written += uint64(n)
	}
	return written
}

// Open syscall
func (k *LinuxKernel) Open(path string, flags enum.OpenFlag, mode uint64) uint64 {
	f, err := k.Fs.OpenFile(path, int(flags), os.FileMode(mode))
	if err != nil {
		return MinusOne
	}
	fd := k.nextfd
	k.nextfd++
	k.Fds[fd] = f
	return uint64(fd)
}

// Read syscall
func (k *LinuxKernel) Read(fd co.Fd, buf co.Obuf, size co.Len) uint64 {
	file, ok := k.Fds[fd]
	if !ok {
		return MinusOne
	}
	tmp := make([]byte, 1024)
	var n uint64
	for i := co.Len(0); i < size; i += 1024 {
		if i+1024 > size {
			tmp = tmp[:size-i]
		}
		count, err := file.Read(tmp)
		if err != nil {
			return MinusOne
		}
		if err := buf.Pack(tmp[:count]); err != nil {
			return MinusOne
		}
		n += uint64(count)
		if count < 1024 {
			break
		}
	}
	return n
}

// Close syscall
func (k *LinuxKernel) Close(fd co.Fd) uint64 {
	file, ok := k.Fds[fd]
	if !ok {
		return MinusOne
	}
	if err := file.Close(); err != nil {
		return MinusOne
	}
	return 0
}

// Stat syscall
func (k *LinuxKernel) Stat(path string, buf co.Obuf) uint64 {
	file, err := k.Fs.Open(path)
	if err != nil {
		return MinusOne
	}
	stat, err := file.Stat()
	if err != nil {
		return MinusOne
	}
	return handleStat(buf, stat, k.U)
}

func handleStat(buf co.Obuf, stat os.FileInfo, u models.Usercorn) uint64 {
	sum := md5.Sum([]byte(stat.Name()))
	ino := binary.BigEndian.Uint64(sum[:])
	s := &LinuxStat64_x86{
		Ino:     ino,
		Size:    stat.Size(),
		Blksize: 1024,
		Mode:    uint32(stat.Mode()),
	}
	return HandleStat(buf, s, u, false)
}

func iovecIter(stream co.Buf, count uint64, bits uint) []Iovec64 {
	res := []Iovec64{}
	st := stream.Struc()
	for i := uint64(0); i < count; i++ {
		if bits == 64 {
			var iovec Iovec64
			st.Unpack(&iovec)
			res = append(res, iovec)
		} else {
			var iv32 Iovec32
			st.Unpack(&iv32)
			res = append(res, Iovec64{
				Base: uint64(iv32.Base),
				Len:  uint64(iv32.Len),
			})
		}
	}
	return res
}
