package zip

import "io"

func openRawReader(r io.ReaderAt, offset int64, header FileHeader) (io.Reader, error) {
	sr := io.NewSectionReader(r, offset, int64(header.CompressedSize64))
	return sr, nil
}
