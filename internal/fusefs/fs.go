package fusefs

import (
	"context"
	"syscall"
	"time"

	"github.com/example/fuses3redispostgres/internal/metadata"
	"github.com/example/fuses3redispostgres/internal/s3io"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type Root struct {
	fs.Inode
	resolver *metadata.Resolver
	reader   *s3io.Reader
	block    int64
	prefetch int64
}

func NewRoot(r *metadata.Resolver, reader *s3io.Reader, block, prefetch int64) *Root {
	return &Root{resolver: r, reader: reader, block: block, prefetch: prefetch}
}

func (r *Root) OnAdd(ctx context.Context) {
	r.NewPersistentInode(ctx, &Dir{name: "files", root: r}, fs.StableAttr{Mode: syscall.S_IFDIR}, true)
	r.NewPersistentInode(ctx, &Dir{name: "by-date", root: r}, fs.StableAttr{Mode: syscall.S_IFDIR}, true)
}

type Dir struct {
	fs.Inode
	name string
	root *Root
}

func (d *Dir) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entries := []fuse.DirEntry{{Name: ".", Mode: syscall.S_IFDIR}, {Name: "..", Mode: syscall.S_IFDIR}, {Name: "README-LIMITED", Mode: syscall.S_IFREG}}
	return fs.NewListDirStream(entries), 0
}

func (d *Dir) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if d.name == "files" {
		obj, err := d.root.resolver.Resolve(ctx, name)
		if err != nil {
			return nil, syscall.ENOENT
		}
		inode := d.NewPersistentInode(ctx, &File{obj: obj, root: d.root}, fs.StableAttr{Mode: syscall.S_IFREG})
		out.SetAttrTimeout(2 * time.Second)
		return inode, 0
	}
	return nil, syscall.ENOENT
}

type File struct {
	fs.Inode
	obj  metadata.Object
	root *Root
}

func (f *File) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = syscall.S_IFREG | 0444
	out.Size = uint64(f.obj.Size)
	return 0
}

func (f *File) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, fuse.FOPEN_DIRECT_IO, 0
}

func (f *File) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	start, end := s3io.AlignRange(off, int64(len(dest)), f.root.block, f.root.prefetch)
	buf, err := f.root.reader.GetRange(ctx, f.obj.Bucket, f.obj.Key, start, end)
	if err != nil {
		return nil, syscall.EIO
	}
	shift := off - start
	if shift >= int64(len(buf)) {
		return fuse.ReadResultData(nil), 0
	}
	max := shift + int64(len(dest))
	if max > int64(len(buf)) {
		max = int64(len(buf))
	}
	return fuse.ReadResultData(buf[shift:max]), 0
}
