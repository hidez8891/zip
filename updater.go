// Copyright 2016 hidez8891. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"io/ioutil"
	"os"
)

type Updater struct {
	path    string
	r       *ReadCloser
	File    []*File
	Comment string
}

// Open exist zip file for editing.
func OpenUpdater(path string) (*Updater, error) {
	r, err := OpenReader(path)
	if err != nil {
		return nil, err
	}

	u := &Updater{
		path:    path,
		r:       r,
		File:    r.File,
		Comment: r.Comment,
	}

	return u, nil
}

// Save write all changes to new zip file and close file.
func (u *Updater) SaveAs(newpath string) error {
	newfile, err := os.Create(newpath)
	if err != nil {
		return err
	}

	defer func() {
		if newfile != nil {
			os.Remove(newpath)
		}
	}()

	// copy & write
	w := NewWriter(newfile)
	for _, file := range u.File {
		if err := w.addFile(file); err != nil {
			newfile.Close()
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}

	// close file
	if err := newfile.Close(); err != nil {
		return err
	}
	newfile = nil
	if err := u.r.Close(); err != nil {
		return err
	}
	u.r = nil

	return nil
}

// Save write all changes to current zip file and close file.
func (u *Updater) Save() error {
	tmpfile, err := ioutil.TempFile("", u.r.f.Name())
	if err != nil {
		return err
	}
	tmpfile.Close()
	tmpname := tmpfile.Name()

	// copy & write & close file
	if err := u.SaveAs(tmpname); err != nil {
		return err
	}

	// move & overwrite
	backuppath := u.path + ".bak"
	if err := os.Rename(u.path, backuppath); err != nil {
		return err
	}
	if err := os.Rename(tmpname, u.path); err != nil {
		return err
	}
	os.Remove(backuppath)

	return nil
}

// Close discard all changes and close file.
// if
func (u *Updater) Close() error {
	if u.r != nil {
		return u.r.Close()
	}
	return nil
}
