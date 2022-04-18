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
	zr    *Reader
	files []*updaterFile
}

// NewUpdater returns a new Updater reading from r, which is assumed to
// have the given size in bytes.
func NewUpdater(r io.ReaderAt, size int64) (*Updater, error) {
	zr, err := NewReader(r, size)
	if err != nil {
		return nil, err
	}

	zu := &Updater{
		zr: zr,
	}
	if err := zu.initFiles(); err != nil {
		return nil, err
	}

	return zu, nil
}

// Open returns a fs.File that provides access to the contents of name's file.
func (z *Updater) Open(name string) (fs.File, error) {
	i := z.findFileIndex(name)
	if i == -1 {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	file := z.files[i]
	if file.existInReader {
		return z.zr.Open(name)
	}

	rb := bytes.NewReader(file.compressedData)
	return openReader(rb, 0, file.header)
}

// Create returns a WriteCloser to which the file contents should be written.
// The file's contents must be written to the io.WriteCloser and closed
// before the next call to Create, SaveAs, Discard or Close.
func (z *Updater) Create(name string) (io.WriteCloser, error) {
	header := &FileHeader{
		Name:   name,
		Method: Deflate,
	}
	bw := new(bytes.Buffer)

	w, err := createWriter(bw, header)
	if err != nil {
		return nil, err
	}

	wc := &fileUpdaterWriteCloser{
		w:      w,
		bw:     bw,
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
	i := z.findFileIndex(oldName)
	if i == -1 {
		return &fs.PathError{Op: "rename", Path: oldName, Err: fs.ErrNotExist}
	}
	if j := z.findFileIndex(newName); j > -1 {
		return fmt.Errorf("invalid duplicate file name: %q", newName)
	}

	file := z.files[i]
	if file.existInReader {
		fr, err := openRawReader(z.zr.r, file.dataOffset, file.header)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, fr); err != nil {
			return err
		}

		if err := z.Delete(oldName); err != nil {
			return err
		}

		header := file.header
		header.Name = newName

		file2 := &updaterFile{
			existInReader:  false,
			header:         header,
			compressedData: buf.Bytes(),
		}
		z.files = append(z.files, file2)
		return nil
	}

	return fmt.Errorf("unimplemented")
}

// Delete deletes the file.
func (z *Updater) Delete(name string) error {
	i := z.findFileIndex(name)
	if i == -1 {
		return &fs.PathError{Op: "delete", Path: name, Err: fs.ErrNotExist}
	}

	z.files = append(z.files[:i], z.files[i+1:]...)
	return nil
}

// SaveAs saves the updated zip file to w.
func (z *Updater) SaveAs(w io.Writer) error {
	ow := NewWriter(w)
	defer ow.Close()

	for _, f := range z.files {
		var fr io.Reader
		if f.existInReader {
			var err error
			fr, err = openRawReader(z.zr.r, f.dataOffset, f.header)
			if err != nil {
				return err
			}
		} else {
			fr = bytes.NewReader(f.compressedData)
		}

		fw, err := ow.CreateRaw(&f.header)
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
	z.zr = nil
	z.files = nil
	return nil
}

// Close is an alias for Discard.
func (z *Updater) Close() error {
	return z.Discard()
}

// Files returns a list of file information in the zip file.
func (z *Updater) Files() []fs.FileInfo {
	fis := make([]fs.FileInfo, len(z.files))
	for i, f := range z.files {
		fis[i] = f.header.FileInfo()
	}
	return fis
}

func (z *Updater) initFiles() error {
	z.zr.initFileList()

	z.files = make([]*updaterFile, len(z.zr.File))
	for i, f := range z.zr.File {
		offset, err := f.findBodyOffset()
		if err != nil {
			return err
		}
		offset += f.headerOffset

		z.files[i] = &updaterFile{
			existInReader: true,
			header:        f.FileHeader,
			dataOffset:    offset,
		}
	}

	return nil
}

func (z *Updater) findFileIndex(name string) int {
	i := slices.IndexFunc(z.files, func(f *updaterFile) bool {
		return f.header.Name == name
	})
	return i
}

type updaterFile struct {
	existInReader  bool
	header         FileHeader
	dataOffset     int64
	compressedData []byte
}

type fileUpdaterWriteCloser struct {
	w      io.WriteCloser
	bw     *bytes.Buffer
	header *FileHeader
	parent *Updater
}

func (w *fileUpdaterWriteCloser) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w *fileUpdaterWriteCloser) Close() error {
	if err := w.w.Close(); err != nil {
		return err
	}

	i := w.parent.findFileIndex(w.header.Name)
	if i > -1 {
		w.parent.files = append(w.parent.files[:i], w.parent.files[i+1:]...)
	}

	file := &updaterFile{
		existInReader:  false,
		header:         *w.header,
		compressedData: w.bw.Bytes(),
	}
	w.parent.files = append(w.parent.files, file)

	return nil
}
