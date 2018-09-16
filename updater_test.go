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

	// check file
	compareContents(t, z, testcase)
}

func TestUpdaterAddFile(t *testing.T) {
	testcase := make([]ZipTestFile, len(updateTest.File))
	copy(testcase, updateTest.File)
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
	testcase = append(testcase, testfile)
	compareContents(t, z, testcase)
}

func TestUpdaterUpdateFile(t *testing.T) {
	testcase := make([]ZipTestFile, len(updateTest.File))
	copy(testcase, updateTest.File)
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
	for i := range testcase {
		if testcase[i].Name == testfile.Name {
			testcase[i] = testfile
		}
	}
	compareContents(t, z, testcase)
}

func TestUpdaterSaveAsFile(t *testing.T) {
	testcase := make([]ZipTestFile, len(updateTest.File))
	copy(testcase, updateTest.File)
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
	wc, err := z.Create(addfile.Name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wc.Write(addfile.Content); err != nil {
		t.Fatal(err)
	}
	if err := wc.Close(); err != nil {
		t.Fatal(err)
	}

	// update file
	wc, err = z.Update(updatefile.Name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wc.Write(updatefile.Content); err != nil {
		t.Fatal(err)
	}
	if err := wc.Close(); err != nil {
		t.Fatal(err)
	}

	// save
	wdump := new(bytes.Buffer)
	if err := z.SaveAs(wdump); err != nil {
		t.Fatal(err)
	}

	// check file
	zr, err := NewUpdater(bytes.NewReader(wdump.Bytes()), int64(wdump.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for i := range testcase {
		if testcase[i].Name == updatefile.Name {
			testcase[i] = updatefile
		}
	}
	testcase = append(testcase, addfile)
	compareContents(t, zr, testcase)
}

func TestUpdaterComment(t *testing.T) {
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

	// update comment
	expected := "new updater comment"
	z.Comment = expected

	// save
	wdump := new(bytes.Buffer)
	if err := z.SaveAs(wdump); err != nil {
		t.Fatal(err)
	}

	// check
	zr, err := NewReader(bytes.NewReader(wdump.Bytes()), int64(wdump.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if zr.Comment != expected {
		t.Fatalf("zip comment=%q, want %q", zr.Comment, expected)
	}
}

func compareContents(t *testing.T, z *Updater, testcase []ZipTestFile) {
	t.Helper()

	files := z.Files()
	if len(testcase) != len(files) {
		t.Fatalf("file count=%d, want %d", len(files), len(testcase))
	}
	for i, ztf := range testcase {
		if files[i].Name != ztf.Name {
			t.Errorf("name=%q, want %q", files[i].Name, ztf.Name)
		}
		compareContent(t, z, ztf)
	}
}

func compareContent(t *testing.T, z *Updater, ztf ZipTestFile) {
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
