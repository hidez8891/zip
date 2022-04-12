package zip

import (
	"io"
	"io/fs"
)

// Updater implements a zip file updater.
type Updater struct {
}

// NewUpdater returns a new Updater reading from r, which is assumed to
// have the given size in bytes.
func NewUpdater(r io.ReaderAt, size int64) (*Updater, error) {
}

// Open returns a fs.File that provides access to the contents of name's file.
func (z *Updater) Open(name string) (fs.File, error) {
}

// Create returns a WriteCloser to which the file contents should be written.
func (z *Updater) Create(name string) (io.WriteCloser, error) {
}

// Update returns a WriteCloser to which the file contents should be overwritten.
func (z *Updater) Update(name string) (io.WriteCloser, error) {
}

// Rename changes the file name.
func (z *Updater) Rename(oldName, newName string) error {
}

// Delete deletes the file.
func (z *Updater) Delete(name string) error {
}

// SaveAs saves the updated zip file to w.
func (z *Updater) SaveAs(w io.Writer) error {
}

// Discard discards the changes and ends editing.
func (z *Updater) Discard() error {
}

// Close is an alias for Discard.
func (z *Updater) Close() error {
	return z.Discard()
}

// Files returns a list of file information in the zip file.
func (z *Updater) Files() ([]fs.FileInfo, error) {
}
