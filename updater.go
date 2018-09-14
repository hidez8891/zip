// Copyright 2018 hidez8891. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	bytesEX "github.com/hidez8891/zip/internal/bytes"
)

// WriteWriterAt is the interface that groups the basic Write and WriteAt methods.
type WriteWriterAt interface {
	io.Writer
	io.WriterAt
}

// A WriteCloser implements the io.WriteCloser
type WriteCloser struct {
	writer io.Writer
	closer io.Closer
}

// Write implements the io.WriteCloser interface.
func (w *WriteCloser) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}

// Close implements the io.WriteCloser interface.
func (w *WriteCloser) Close() error {
	return w.closer.Close()
}

// Updater provides editing of zip files.
type Updater struct {
	files   []string
	headers map[string]*FileHeader
	entries map[string]*bytesEX.BufferAt
	r       *Reader
}

// NewUpdater returns a new Updater from r and size.
func NewUpdater(r io.ReaderAt, size int64) (*Updater, error) {
	zr, err := NewReader(r, size)
	if err != nil {
		return nil, err
	}

	files := make([]string, len(zr.File))
	headers := make(map[string]*FileHeader, len(zr.File))
	for i, zf := range zr.File {
		files[i] = zf.Name
		headers[zf.Name] = &zf.FileHeader
	}

	return &Updater{
		files:   files,
		headers: headers,
		entries: make(map[string]*bytesEX.BufferAt),
		r:       zr,
	}, nil
}

// Files returns a FileHeader list.
func (u *Updater) Files() []*FileHeader {
	files := make([]*FileHeader, len(u.files))
	for i, name := range u.files {
		files[i] = u.headers[name]
	}
	return files
}

// Open returns a ReadCloser that provides access to the File's contents.
func (u *Updater) Open(name string) (io.ReadCloser, error) {
	if _, ok := u.headers[name]; !ok {
		return nil, errors.New("File not found")
	}

	if buf, ok := u.entries[name]; ok {
		b := buf.Bytes()
		z, err := NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			return nil, err
		}
		return z.File[0].Open()
	}

	for _, zf := range u.r.File {
		if zf.Name == name {
			return zf.Open()
		}
	}
	return nil, errors.New("internal error: name not found")
}

// Create returns a Writer to which the file contents should be written.
func (u *Updater) Create(name string) (io.WriteCloser, error) {
	if _, ok := u.headers[name]; ok {
		return nil, errors.New("invalid duplicate file name")
	}

	u.entries[name] = new(bytesEX.BufferAt)
	z := NewWriter(u.entries[name])

	w, err := z.Create(name)
	if err != nil {
		return nil, err
	}
	u.files = append(u.files, name)
	u.headers[name] = z.dir[0].FileHeader

	wc := &WriteCloser{
		writer: w,
		closer: z,
	}
	return wc, nil
}

// Update returns a Writer to which the file contents should be overwritten.
func (u *Updater) Update(name string) (io.WriteCloser, error) {
	if _, ok := u.headers[name]; !ok {
		return nil, errors.New("not found file name")
	}
	useDataDescriptor := u.headers[name].Flags&FlagDataDescriptor != 0

	u.entries[name] = new(bytesEX.BufferAt)
	z := NewWriter(u.entries[name])

	w, err := z.CreateHeader(u.headers[name])
	if err != nil {
		return nil, err
	}
	if !useDataDescriptor {
		z.dir[0].FileHeader.Flags &^= FlagDataDescriptor
	}
	u.headers[name] = z.dir[0].FileHeader

	wc := &WriteCloser{
		writer: w,
		closer: z,
	}
	return wc, nil
}

// SaveAs saves the changes to w.
// If data descriptor is not used, w must implement io.WriterAt.
func (u *Updater) SaveAs(w io.Writer) error {
	z := NewWriter(w)

	for _, name := range u.files {
		offset := z.cw.count

		fh := u.headers[name]
		if err := writeHeader(z.cw, fh); err != nil {
			return err
		}
		z.dir = append(z.dir, &header{
			FileHeader: fh,
			offset:     uint64(offset),
		})

		if entry, ok := u.entries[name]; ok {
			// write new file
			size := int64(fh.CompressedSize64)
			if fh.Flags&FlagDataDescriptor != 0 {
				if fh.isZip64() {
					size += dataDescriptor64Len
				} else {
					size += dataDescriptorLen
				}
			}
			bodyOffset := z.cw.count - offset
			bodyEnd := bodyOffset + size

			bw := bytes.NewReader(entry.Bytes()[bodyOffset:bodyEnd])
			if _, err := io.Copy(z.cw, bw); err != nil {
				return err
			}
		} else {
			// write zip's content
			var zfile *File
			for _, zf := range u.r.File {
				if zf.Name == name {
					zfile = zf
				}
			}
			if zfile == nil {
				return fmt.Errorf("internal error: %s is not exist", name)
			}

			size := int64(zfile.CompressedSize64)
			if zfile.Flags&FlagDataDescriptor != 0 {
				if fh.isZip64() {
					size += dataDescriptor64Len
				} else {
					size += dataDescriptorLen
				}
			}
			bodyOffset, err := zfile.findBodyOffset()
			if err != nil {
				return err
			}
			r := io.NewSectionReader(zfile.zipr, zfile.headerOffset+bodyOffset, size)
			if _, err := io.Copy(z.cw, r); err != nil {
				return err
			}
		}
	}

	return z.Close()
}

// Cancel discards the changes and ends editing.
func (u *Updater) Cancel() error {
	u.files = make([]string, 0)
	u.headers = make(map[string]*FileHeader, 0)
	u.entries = make(map[string]*bytesEX.BufferAt, 0)
	u.r = nil
	return nil
}

// Close discards the changes and ends editing.
func (u *Updater) Close() error {
	return u.Cancel()
}
