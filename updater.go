// Copyright 2018 hidez8891. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"errors"
	"io"
)

// Updater provides editing of zip files.
type Updater struct {
	path    string
	files   []string
	headers map[string]*FileHeader
	r       *ReadCloser
}

// NewUpdater returns a new Updater from path.
func NewUpdater(path string) (*Updater, error) {
	r, err := OpenReader(path)
	if err != nil {
		return nil, err
	}

	files := make([]string, len(r.File))
	headers := make(map[string]*FileHeader, len(r.File))
	for i, zf := range r.File {
		files[i] = zf.Name
		headers[zf.Name] = &zf.FileHeader
	}

	return &Updater{
		path:    path,
		files:   files,
		headers: headers,
		r:       r,
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

	for _, zf := range u.r.File {
		if zf.Name == name {
			return zf.Open()
		}
	}
	return nil, errors.New("internal error: name not found")
}

// Create returns a Writer to which the file contents should be written.
func (u *Updater) Create(name string) (io.Writer, error) {
	return nil, errors.New("internal error: Unimplemented")
}

// Update returns a Writer to which the file contents should be overwritten.
func (u *Updater) Update(name string) (io.Writer, error) {
	return nil, errors.New("internal error: Unimplemented")
}

// Save saves the changes and ends editing.
func (u *Updater) Save() error {
	err := u.r.Close()
	if err != nil {
		return err
	}
	return errors.New("internal error: Unimplemented")
}

// SaveAs saves the changes to path and ends editing.
func (u *Updater) SaveAs(path string) error {
	err := u.r.Close()
	if err != nil {
		return err
	}
	return errors.New("internal error: Unimplemented")
}

// Cancel discards the changes and ends editing.
func (u *Updater) Cancel() error {
	return u.r.Close()
}

// Close discards the changes and ends editing.
func (u *Updater) Close() error {
	return u.Cancel()
}
