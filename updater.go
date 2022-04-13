package zip

import (
	"bytes"
	"io"
	"io/fs"
)

// Updater implements a zip file updater.
type Updater struct {
	zr    *Reader
	zw    *Writer
	buf   *bytes.Buffer
	files []fs.FileInfo
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
	return z.zr.Open(name)
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

// Update returns a WriteCloser to which the file contents should be overwritten.
func (z *Updater) Update(name string) (io.WriteCloser, error) {
	return nil, nil
}

// Rename changes the file name.
func (z *Updater) Rename(oldName, newName string) error {
	return nil
}

// Delete deletes the file.
func (z *Updater) Delete(name string) error {
	return nil
}

// SaveAs saves the updated zip file to w.
func (z *Updater) SaveAs(w io.Writer) error {
	if err := z.zw.Close(); err != nil {
		return err
	}

	ow := NewWriter(w)
	for _, f := range z.zr.File {
		header := f.FileHeader
		fw, err := ow.CreateRaw(&header)
		if err != nil {
			return err
		}
		fr, err := f.OpenRaw()
		if err != nil {
			return err
		}
		io.Copy(fw, fr)
	}

	br, err := NewReader(bytes.NewReader(z.buf.Bytes()), int64(z.buf.Len()))
	if err != nil {
		return err
	}
	for _, f := range br.File {
		header := f.FileHeader
		fw, err := ow.CreateRaw(&header)
		if err != nil {
			return err
		}
		fr, err := f.OpenRaw()
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
	for i, f := range z.zr.File {
		name := f.FileHeader.Name
		e := z.zr.openLookup(name)
		z.files[i] = e.stat()
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
	if fw, ok := w.w.(*fileWriter); ok {
		if err := fw.close(); err != nil {
			return err
		}
		info := headerFileInfo{w.header}
		w.parent.files = append(w.parent.files, info)
	} else {
		info := &fileListEntry{
			name:  w.header.Name,
			file:  nil,
			isDir: true,
		}
		w.parent.files = append(w.parent.files, info)
	}
	return nil
}
