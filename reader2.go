package zip

import (
	"hash"
	"hash/crc32"
	"io"
	"io/fs"
)

func openReader(r io.ReaderAt, offset int64, header FileHeader) (fs.File, error) {
	sr := io.NewSectionReader(r, offset, int64(header.CompressedSize64))
	dcomp := decompressor(header.Method)
	if dcomp == nil {
		return nil, ErrAlgorithm
	}
	var rc io.ReadCloser = dcomp(sr)
	cr := &checksumReadCloser{
		rc:     rc,
		hash:   crc32.NewIEEE(),
		header: header,
	}
	return cr, nil
}

func openRawReader(r io.ReaderAt, offset int64, header FileHeader) (io.Reader, error) {
	sr := io.NewSectionReader(r, offset, int64(header.CompressedSize64))
	return sr, nil
}

type checksumReadCloser struct {
	rc     io.ReadCloser
	hash   hash.Hash32
	header FileHeader
	nread  uint64 // number of bytes read so far
	err    error  // sticky error
}

func (r *checksumReadCloser) Stat() (fs.FileInfo, error) {
	return headerFileInfo{&r.header}, nil
}

func (r *checksumReadCloser) Read(b []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}
	n, err = r.rc.Read(b)
	r.hash.Write(b[:n])
	r.nread += uint64(n)
	if err == nil {
		return
	}
	if err == io.EOF {
		if r.nread != r.header.UncompressedSize64 {
			return 0, io.ErrUnexpectedEOF
		}
		if r.header.CRC32 != 0 && r.hash.Sum32() != r.header.CRC32 {
			err = ErrChecksum
		}
	}
	r.err = err
	return
}

func (r *checksumReadCloser) Close() error { return r.rc.Close() }
