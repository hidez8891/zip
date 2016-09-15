// Copyright 2016 hidez8891. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var updateTests = []ZipTest{
	{
		Name:    "test.zip",
		Comment: "This is a zipfile comment.",
		File: []ZipTestFile{
			{
				Name:    "test.txt",
				Content: []byte("This is a test text file.\n"),
				Mtime:   "09-05-10 12:12:02",
				Mode:    0644,
			},
			{
				Name:  "gophercolor16x16.png",
				File:  "gophercolor16x16.png",
				Mtime: "09-05-10 15:52:58",
				Mode:  0644,
			},
		},
	},
}

var updateAppendFiles = []WriteTest{
	{
		Name:   "foo",
		Data:   []byte("Rabbits, guinea pigs, gophers, marsupial rats, and quolls."),
		Method: Store,
		Mode:   0666,
	},
}

func TestUpdaterOnlyCopy(t *testing.T) {
	for _, zt := range updateTests {
		updaterOnlyCopy(t, zt)
	}
}

func updaterOnlyCopy(t *testing.T, zt ZipTest) {
	testfile := filepath.Join("testdata", zt.Name)
	z, err := OpenUpdater(testfile)
	if err != nil {
		t.Fatalf("%s open failed: %v", zt.Name, err)
	}

	tmpfile, err := ioutil.TempFile("", "test_zip_updater_")
	if err != nil {
		t.Fatalf("tempfile create failed: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	if err := z.SaveAs(tmpfile.Name()); err != nil {
		t.Fatalf("%s save to %s failed: %v", zt.Name, tmpfile.Name(), err)
	}

	if sameFileCheck(testfile, tmpfile.Name()) == false {
		t.Fatalf("%s not same to %s: copy failed", zt.Name, tmpfile.Name())
	}
}

func sameFileCheck(path1, path2 string) bool {
	// zip.Writer isn't conform to the zip's specifications.
	//   (1) Local File Header's extra fields are ignored.
	//       So, Central Directory Header's is written insted.
	//   (2) zip's comment is ignored.
	//
	// inState, _ := os.Stat(path1)
	// outState, _ := os.Stat(path2)
	// return os.SameFile(inState, outState)

	z1, _ := OpenReader(path1)
	defer z1.Close()
	z2, _ := OpenReader(path2)
	defer z2.Close()

	if len(z1.File) != len(z2.File) {
		return false
	}

	for i := range z1.File {
		f1 := z1.File[i]
		f2 := z2.File[i]

		if f1.Name != f2.Name {
			return false
		}

		r1, _ := f1.bodyReader()
		r2, _ := f2.bodyReader()
		if sameReader(r1, r2) == false {
			return false
		}
	}

	return true
}

func sameReader(r1, r2 io.Reader) bool {
	const chunksize = 1024

	for {
		b1 := make([]byte, chunksize)
		_, err1 := r1.Read(b1)

		b2 := make([]byte, chunksize)
		_, err2 := r2.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true
			}
			if err1 == io.EOF || err2 == io.EOF {
				return false
			}

			// happen fatal error
			return false
		}

		if bytes.Equal(b1, b2) == false {
			return false
		}
	}
}

func TestUpdaterAppendFile(t *testing.T) {
	for _, zt := range updateTests {
		updaterAppendFile(t, zt)
	}
}

func updaterAppendFile(t *testing.T, zt ZipTest) {
	testfile := filepath.Join("testdata", zt.Name)
	z, err := OpenUpdater(testfile)
	if err != nil {
		t.Fatalf("%s open failed: %v", zt.Name, err)
	}

	for _, file := range updateAppendFiles {
		w, err := z.AppendFile(file.Name, true)
		if err != nil {
			t.Fatalf("%s failed append file header: %v", zt.Name, err)
		}
		if _, err := io.Copy(w, bytes.NewReader(file.Data)); err != nil {
			t.Fatalf("%s failed append file data: %v", zt.Name, err)
		}
	}

	tmpfile, err := ioutil.TempFile("", "test_zip_updater_")
	if err != nil {
		t.Fatalf("tempfile create failed: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	if err := z.SaveAs(tmpfile.Name()); err != nil {
		t.Fatalf("%s save to %s failed: %v", zt.Name, tmpfile.Name(), err)
	}

	r, err := OpenReader(tmpfile.Name())
	if err != nil {
		t.Fatalf("%s open failed: %v", tmpfile.Name(), err)
	}

	for _, file := range zt.File {
		var found bool
		for _, f := range r.File {
			if file.Name == f.Name {
				found = true
				break
			}
		}

		if found == false {
			t.Fatalf("%s does not have %s", tmpfile.Name(), file.Name)
		}
	}

	for _, file := range updateAppendFiles {
		var found bool
		for _, f := range r.File {
			if file.Name == f.Name {
				found = true
				break
			}
		}

		if found == false {
			t.Fatalf("%s does not have %s", tmpfile.Name(), file.Name)
		}
	}
}
