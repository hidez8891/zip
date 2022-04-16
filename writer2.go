package zip

import (
	"hash/crc32"
	"io"
	"strings"
)

func createWriter(w io.Writer, fh *FileHeader) (io.Writer, error) {
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
		ow io.Writer
		fw *fileWriter
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

		ow = dirWriter{}
	} else {
		fh.Flags |= 0x8 // we will write a data descriptor

		fw = &fileWriter{
			zipw:      w,
			compCount: &countWriter{w: w},
			crc32:     crc32.NewIEEE(),
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

func createRawWriter(w io.Writer, fh *FileHeader) (io.Writer, error) {
	fh.CompressedSize = uint32(min64(fh.CompressedSize64, uint32max))
	fh.UncompressedSize = uint32(min64(fh.UncompressedSize64, uint32max))

	h := &header{
		FileHeader: fh,
		offset:     0,
		raw:        true,
	}

	if strings.HasSuffix(fh.Name, "/") {
		return dirWriter{}, nil
	}

	fw := &fileWriter{
		header: h,
		zipw:   w,
	}
	return fw, nil
}
