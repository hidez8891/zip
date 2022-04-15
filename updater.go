package zip

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

	"golang.org/x/exp/slices"
)

// Updater implements a zip file updater.
type Updater struct {
	zr      *Reader
	zw      *Writer
	buf     *bytes.Buffer
	files   []fs.FileInfo
	headers map[string]fileRWHeader
}

// NewUpdater returns a new Updater reading from r, which is assumed to
// have the given size in bytes.
func NewUpdater(r io.ReaderAt, size int64) (*Updater, error) {
	zr, err := NewReader(r, size)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	zw := NewWriter(buf)
	zu := &Updater{
		zr:  zr,
		zw:  zw,
		buf: buf,
	}
	if err := zu.initFiles(); err != nil {
		return nil, err
	}

	return zu, nil
}

// Open returns a fs.File that provides access to the contents of name's file.
func (z *Updater) Open(name string) (fs.File, error) {
	header, ok := z.headers[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	if header.existInReader {
		return z.zr.Open(name)
	}

	return nil, fmt.Errorf("unimplemented")
}

// Create returns a WriteCloser to which the file contents should be written.
// The file's contents must be written to the io.WriteCloser and closed
// before the next call to Create, SaveAs, Discard or Close.
func (z *Updater) Create(name string) (io.WriteCloser, error) {
	header := &FileHeader{
		Name:   name,
		Method: Deflate,
	}

	w, err := z.zw.CreateHeader(header)
	if err != nil {
		return nil, err
	}

	wc := &fileWriteCloser{
		w:      w,
		header: header,
		parent: z,
	}
	return wc, nil
}

// Update returns a fs.File that provides access to the contents of name's file
// and a WriteCloser to which the file contents should be overwritten.
func (z *Updater) Update(name string) (fs.File, io.WriteCloser, error) {
	r, err := z.Open(name)
	if err != nil {
		return nil, nil, err
	}

	w, err := z.Create(name)
	if err != nil {
		r.Close()
		return nil, nil, err
	}

	return r, w, nil
}

// Rename changes the file name.
func (z *Updater) Rename(oldName, newName string) error {
	return nil
}

// Delete deletes the file.
func (z *Updater) Delete(name string) error {
	if _, ok := z.headers[name]; !ok {
		return &fs.PathError{Op: "delete", Path: name, Err: fs.ErrNotExist}
	}

	i := slices.IndexFunc(z.files, func(f fs.FileInfo) bool {
		return f.Name() == name
	})
	if i == -1 {
		return fmt.Errorf("BUG: %s is not exist in the temporary or read file.", name)
	}
	z.files = append(z.files[:i], z.files[i+1:]...)
	delete(z.headers, name)

	return nil
}

// SaveAs saves the updated zip file to w.
func (z *Updater) SaveAs(w io.Writer) error {
	if err := z.zw.Close(); err != nil {
		return err
	}

	ow := NewWriter(w)
	defer ow.Close()

	br, err := NewReader(bytes.NewReader(z.buf.Bytes()), int64(z.buf.Len()))
	if err != nil {
		return err
	}

	for _, f := range z.files {
		name := f.Name()
		header := z.headers[name]

		fw, err := ow.CreateRaw(header.FileHeader)
		if err != nil {
			return err
		}

		var zr *Reader
		if header.existInReader {
			zr = z.zr
		} else {
			zr = br
		}

		index := -1
		headerOffset := int64(-1)
		for i, zf := range zr.File {
			if zf.Name != name {
				continue
			}
			if headerOffset < zf.headerOffset {
				headerOffset = zf.headerOffset
				index = i
			}
		}
		if index == -1 {
			return fmt.Errorf("BUG: %s is not found", name)
		}
		fr, err := zr.File[index].OpenRaw()
		if err != nil {
			return err
		}

		_, err = io.Copy(fw, fr)
		if err != nil {
			return err
		}
	}

	if err := ow.Close(); err != nil {
		return err
	}
	return nil
}

// Discard discards the changes and ends editing.
func (z *Updater) Discard() error {
	z.zw.Close()

	z.zr = nil
	z.zw = nil
	z.buf = nil
	z.files = nil
	z.headers = nil
	return nil
}

// Close is an alias for Discard.
func (z *Updater) Close() error {
	return z.Discard()
}

// Files returns a list of file information in the zip file.
func (z *Updater) Files() []fs.FileInfo {
	return z.files[:]
}

func (z *Updater) initFiles() error {
	z.zr.initFileList()

	z.files = make([]fs.FileInfo, len(z.zr.File))
	z.headers = make(map[string]fileRWHeader)
	for i, f := range z.zr.File {
		name := f.FileHeader.Name
		e := z.zr.openLookup(name)
		z.files[i] = e.stat()
		z.headers[name] = fileRWHeader{
			FileHeader:    &f.FileHeader,
			existInReader: true,
		}
	}

	return nil
}

type fileWriteCloser struct {
	w      io.Writer
	header *FileHeader
	parent *Updater
}

func (w *fileWriteCloser) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w *fileWriteCloser) Close() error {
	var info fs.FileInfo
	if fw, ok := w.w.(*fileWriter); ok {
		if err := fw.close(); err != nil {
			return err
		}
		info = headerFileInfo{w.header}
	} else {
		info = &fileListEntry{
			name:  w.header.Name,
			file:  nil,
			isDir: true,
		}
	}

	if _, ok := w.parent.headers[w.header.Name]; ok {
		i := slices.IndexFunc(w.parent.files, func(f fs.FileInfo) bool {
			return f.Name() == w.header.Name
		})
		if i == -1 {
			return fmt.Errorf("BUG: %s is not exist in the temporary or read file.", w.header.Name)
		}
		w.parent.files = append(w.parent.files[:i], w.parent.files[i+1:]...)
	}

	w.parent.files = append(w.parent.files, info)
	w.parent.headers[w.header.Name] = fileRWHeader{
		FileHeader:    w.header,
		existInReader: false,
	}
	return nil
}

type fileRWHeader struct {
	*FileHeader
	existInReader bool
}
