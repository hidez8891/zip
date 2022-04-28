package zip

import (
	"errors"
	"hash/crc32"
	"io"
	"strings"
)

func createWriter(w io.Writer, fh *FileHeader) (io.WriteCloser, error) {
	utf8Valid1, utf8Require1 := detectUTF8(fh.Name)
	utf8Valid2, utf8Require2 := detectUTF8(fh.Comment)
	switch {
	case fh.NonUTF8:
		fh.Flags &^= 0x800
	case (utf8Require1 || utf8Require2) && (utf8Valid1 && utf8Valid2):
		fh.Flags |= 0x800
	}

	fh.CreatorVersion = fh.CreatorVersion&0xff00 | zipVersion20 // preserve compatibility byte
	fh.ReaderVersion = zipVersion20

	if !fh.Modified.IsZero() {
		fh.ModifiedDate, fh.ModifiedTime = timeToMsDosTime(fh.Modified)
	}

	var (
		ow io.WriteCloser
		fw *fileWriteCloser
	)
	h := &header{
		FileHeader: fh,
		offset:     0,
	}

	if strings.HasSuffix(fh.Name, "/") {
		fh.Method = Store
		fh.Flags &^= 0x8 // we will not write a data descriptor

		// Explicitly clear sizes as they have no meaning for directories.
		fh.CompressedSize = 0
		fh.CompressedSize64 = 0
		fh.UncompressedSize = 0
		fh.UncompressedSize64 = 0

		ow = dirWriteCloser{}
	} else {
		fw = &fileWriteCloser{
			fileWriter{
				zipw:      &writeOnlyReadWriter{w},
				compCount: &countWriter{w: w},
				crc32:     crc32.NewIEEE(),
			},
		}
		comp := compressor(fh.Method)
		if comp == nil {
			return nil, ErrAlgorithm
		}
		var err error
		fw.comp, err = comp(fw.compCount)
		if err != nil {
			return nil, err
		}
		fw.rawCount = &countWriter{w: fw.comp}
		fw.header = h
		ow = fw
	}
	return ow, nil
}

type dirWriteCloser struct {
	dirWriter
}

func (dirWriteCloser) Close() error {
	return nil
}

type fileWriteCloser struct {
	fileWriter
}

func (f *fileWriteCloser) Close() error {
	return f.close()
}

type writeOnlyReadWriter struct {
	w io.Writer
}

func (rw *writeOnlyReadWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *writeOnlyReadWriter) Read(b []byte) (int, error) {
	return 0, errors.New("zip: read from write-only data")
}
