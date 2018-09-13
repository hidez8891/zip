// Copyright 2018 hidez8891. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"bytes"
	"io"
	"os"
	"testing"
)

var updateTest = ZipTest{
	Name: "winxp.zip",
	File: []ZipTestFile{
		{
			Name:    "hello",
			Content: []byte("world \r\n"),
		},
		{
			Name:    "dir/bar",
			Content: []byte("foo \r\n"),
		},
		{
			Name:    "dir/empty/",
			Content: []byte{},
		},
		{
			Name:    "readonly",
			Content: []byte("important \r\n"),
		},
	},
}

func TestUpdaterOnlyRead(t *testing.T) {
	testcase := updateTest.File
	file, err := os.Open("testdata/" + updateTest.Name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	st, _ := file.Stat()
	z, err := NewUpdater(file, st.Size())
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()

	files := z.Files()
	if len(testcase) != len(files) {
		t.Fatalf("file count=%d, want %d", len(files), len(testcase))
	}
	for i, ztf := range testcase {
		if files[i].Name != ztf.Name {
			t.Errorf("name=%q, want %q", files[i].Name, ztf.Name)
		}
		compareContents(t, z, ztf)
	}
}

func TestUpdaterAddFile(t *testing.T) {
	testcase := updateTest.File
	testfile := ZipTestFile{
		Name:    "test",
		Content: []byte("text string"),
	}

	file, err := os.Open("testdata/" + updateTest.Name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	st, _ := file.Stat()
	z, err := NewUpdater(file, st.Size())
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()

	// add file
	wc, err := z.Create(testfile.Name)
	if err != nil {
		t.Fatal(err)
	}

	n, err := wc.Write(testfile.Content)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(testfile.Content) {
		t.Fatalf("write size=%d, want %d", n, len(testfile.Content))
	}
	if err := wc.Close(); err != nil {
		t.Fatal(err)
	}

	// check file
	files := z.Files()
	if len(files) != len(testcase)+1 {
		t.Fatalf("file count=%d, want %d", len(files), len(testcase)+1)
	}

	for i, ztf := range testcase {
		if files[i].Name != ztf.Name {
			t.Errorf("name=%q, want %q", files[i].Name, ztf.Name)
		}
		compareContents(t, z, ztf)
	}

	last := files[len(files)-1]
	if last.Name != testfile.Name {
		t.Errorf("name=%q, want %q", last.Name, testfile.Name)
	}
	compareContents(t, z, testfile)
}

func TestUpdaterUpdateFile(t *testing.T) {
	testcase := updateTest.File
	testfile := ZipTestFile{
		Name:    "dir/bar",
		Content: []byte("update string"),
	}

	file, err := os.Open("testdata/" + updateTest.Name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	st, _ := file.Stat()
	z, err := NewUpdater(file, st.Size())
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()

	// update file
	wc, err := z.Update(testfile.Name)
	if err != nil {
		t.Fatal(err)
	}

	n, err := wc.Write(testfile.Content)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(testfile.Content) {
		t.Fatalf("write size=%d, want %d", n, len(testfile.Content))
	}
	if err := wc.Close(); err != nil {
		t.Fatal(err)
	}

	// check file
	files := z.Files()
	if len(files) != len(testcase) {
		t.Fatalf("file count=%d, want %d", len(files), len(testcase))
	}

	for i, ztf := range testcase {
		if files[i].Name != ztf.Name {
			t.Errorf("name=%q, want %q", files[i].Name, ztf.Name)
		}
		if files[i].Name == testfile.Name {
			compareContents(t, z, testfile)
		} else {
			compareContents(t, z, ztf)
		}
	}
}

func TestUpdaterSaveAsFile(t *testing.T) {
	testcase := updateTest.File
	updatefile := ZipTestFile{
		Name:    "dir/bar",
		Content: []byte("update string"),
	}
	addfile := ZipTestFile{
		Name:    "test",
		Content: []byte("text string"),
	}

	file, err := os.Open("testdata/" + updateTest.Name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	st, _ := file.Stat()
	z, err := NewUpdater(file, st.Size())
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()

	// add file
	wc, _ := z.Create(addfile.Name)
	wc.Write(addfile.Content)
	wc.Close()

	// update file
	wc, _ = z.Update(updatefile.Name)
	wc.Write(updatefile.Content)
	wc.Close()

	// save
	wdump := new(bytes.Buffer)
	err = z.SaveAs(wdump)
	if err != nil {
		t.Fatal(err)
	}

	// check file
	zr, err := NewReader(bytes.NewReader(wdump.Bytes()), int64(wdump.Len()))
	if err != nil {
		t.Fatal(err)
	}

	files := zr.File
	if len(files) != len(testcase)+1 {
		t.Fatalf("file count=%d, want %d", len(files), len(testcase)+1)
	}

	for i, ztf := range testcase {
		if files[i].Name != ztf.Name {
			t.Errorf("name=%q, want %q", files[i].Name, ztf.Name)
		}
		if files[i].Name == updatefile.Name {
			compareContents(t, z, updatefile)
		} else {
			compareContents(t, z, ztf)
		}
	}

	last := files[len(files)-1]
	if last.Name != addfile.Name {
		t.Errorf("name=%q, want %q", last.Name, addfile.Name)
	}
	compareContents(t, z, addfile)
}

func compareContents(t *testing.T, z *Updater, ztf ZipTestFile) {
	t.Helper()

	r, err := z.Open(ztf.Name)
	if err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	_, err = io.Copy(&b, r)
	r.Close()
	if err != nil {
		t.Fatal(err)
	}

	buf := b.Bytes()
	if len(buf) != len(ztf.Content) {
		t.Fatalf("filesize len=%d, want %d", len(buf), len(ztf.Content))
	}
	for i, c := range ztf.Content {
		if buf[i] != c {
			t.Fatalf("content[%d]=%q, want %q", i, buf[i], c)
		}
	}
}
