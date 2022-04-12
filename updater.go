package zip

import (
	"io"
	"io/fs"
)

// Updater implements a zip file updater.
type Updater struct {
	zr *Reader
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
	return zu, nil
}

// Open returns a fs.File that provides access to the contents of name's file.
func (z *Updater) Open(name string) (fs.File, error) {
	return z.zr.Open(name)
}

// Create returns a WriteCloser to which the file contents should be written.
func (z *Updater) Create(name string) (io.WriteCloser, error) {
	return nil, nil
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
	return nil
}

// Discard discards the changes and ends editing.
func (z *Updater) Discard() error {
	return nil
}

// Close is an alias for Discard.
func (z *Updater) Close() error {
	return z.Discard()
}

// Files returns a list of file information in the zip file.
func (z *Updater) Files() []fs.FileInfo {
	info := make([]fs.FileInfo, len(z.zr.File))

	for i, f := range z.zr.File {
		zf, err := z.zr.Open(f.Name)
		if err != nil {
			panic(err) // zip.Reader internal error
		}

		info[i], err = zf.Stat()
		if err != nil {
			panic(err) // zip.Reader internal error
		}

		zf.Close()
	}

	return info
}
