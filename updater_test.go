// Copyright 2018 hidez8891. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"bytes"
	"io"
	"os"
	"sort"
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
	// open file
	file, z := testOpenFile(t, "testdata/"+updateTest.Name)
	defer file.Close()
	defer z.Close()

	// check file
	testcase := updateTest.File
	compareContents(t, z, testcase)
}

func TestUpdaterAddFile(t *testing.T) {
	addfile := ZipTestFile{
		Name:    "test",
		Content: []byte("text string"),
	}

	// open file
	file, z := testOpenFile(t, "testdata/"+updateTest.Name)
	defer file.Close()
	defer z.Close()

	// add file
	testAddFile(t, z, addfile)

	// check file
	testcase := make([]ZipTestFile, len(updateTest.File))
	copy(testcase, updateTest.File)
	testcase = append(testcase, addfile)
	compareContents(t, z, testcase)
}

func TestUpdaterUpdateFile(t *testing.T) {
	updatefile := ZipTestFile{
		Name:    "dir/bar",
		Content: []byte("update string"),
	}

	// open file
	file, z := testOpenFile(t, "testdata/"+updateTest.Name)
	defer file.Close()
	defer z.Close()

	// update file
	testUpdateFile(t, z, updatefile)

	// check file
	testcase := make([]ZipTestFile, len(updateTest.File))
	copy(testcase, updateTest.File)
	for i := range testcase {
		if testcase[i].Name == updatefile.Name {
			testcase[i] = updatefile
		}
	}
	compareContents(t, z, testcase)
}

func TestUpdaterSaveAsFile(t *testing.T) {
	updatefile := ZipTestFile{
		Name:    "dir/bar",
		Content: []byte("update string"),
	}
	addfile := ZipTestFile{
		Name:    "test",
		Content: []byte("text string"),
	}

	// open file
	file, z := testOpenFile(t, "testdata/"+updateTest.Name)
	defer file.Close()
	defer z.Close()

	// add & update file
	testAddFile(t, z, addfile)
	testUpdateFile(t, z, updatefile)

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

	testcase := make([]ZipTestFile, len(updateTest.File))
	copy(testcase, updateTest.File)
	for i := range testcase {
		if testcase[i].Name == updatefile.Name {
			testcase[i] = updatefile
		}
	}
	testcase = append(testcase, addfile)
	compareContents(t, zr, testcase)
}

func TestUpdaterComment(t *testing.T) {
	// open file
	file, z := testOpenFile(t, "testdata/"+updateTest.Name)
	defer file.Close()
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

func TestUpdaterReadComment(t *testing.T) {
	filename := "test.zip"
	comment := "This is a zipfile comment."

	// open file
	file, z := testOpenFile(t, "testdata/"+filename)
	defer file.Close()
	defer z.Close()

	// check
	if z.Comment != comment {
		t.Fatalf("zip comment=%q, want %q", z.Comment, comment)
	}
}

func TestUpdaterRenameFile(t *testing.T) {
	addfile := ZipTestFile{
		Name:    "test",
		Content: []byte("text string"),
	}

	oldname1, newname1 := "dir/bar", "dir/abcd"
	oldname2, newname2 := "test", "testing"

	// open file
	file, z := testOpenFile(t, "testdata/"+updateTest.Name)
	defer file.Close()
	defer z.Close()

	// add file
	testAddFile(t, z, addfile)

	// rename file
	if err := z.Rename(oldname1, newname1); err != nil {
		t.Fatal(err)
	}
	if err := z.Rename(oldname2, newname2); err != nil {
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

	testcase := make([]ZipTestFile, len(updateTest.File))
	copy(testcase, updateTest.File)
	for i, zf := range testcase {
		if zf.Name == oldname1 {
			testcase[i] = ZipTestFile{
				Name:    newname1,
				Content: zf.Content,
			}
		}
	}
	testcase = append(testcase, ZipTestFile{
		Name:    newname2,
		Content: addfile.Content,
	})
	compareContents(t, zr, testcase)
}

func TestUpdaterRemoveFile(t *testing.T) {
	name := "dir/bar"

	// open file
	file, z := testOpenFile(t, "testdata/"+updateTest.Name)
	defer file.Close()
	defer z.Close()

	// remove file
	if err := z.Remove(name); err != nil {
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

	testcase := make([]ZipTestFile, 0)
	for _, zf := range updateTest.File {
		if zf.Name != name {
			testcase = append(testcase, zf)
		}
	}
	compareContents(t, zr, testcase)
}

func TestUpdaterSortFile(t *testing.T) {
	names := make([]string, len(updateTest.File))
	for i, tf := range updateTest.File {
		names[i] = tf.Name
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] > names[j] // reverse sort
	})

	// open file
	file, z := testOpenFile(t, "testdata/"+updateTest.Name)
	defer file.Close()
	defer z.Close()

	// sort
	err := z.Sort(func(s []string) []string {
		sort.Slice(s, func(i, j int) bool {
			return s[i] > s[j]
		})
		return s
	})
	if err != nil {
		t.Fatal(err)
	}

	// sort error
	if err := z.Sort(func(s []string) []string { return make([]string, 0) }); err == nil {
		t.Fatalf("need raise error")
	}
	if err := z.Sort(func(s []string) []string { return make([]string, len(s)) }); err == nil {
		t.Fatalf("need raise error")
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

	testcase := make([]ZipTestFile, len(names))
	for _, zf := range updateTest.File {
		for i, name := range names {
			if zf.Name == name {
				testcase[i] = zf
				break
			}
		}
	}
	compareContents(t, zr, testcase)
}

func testOpenFile(t *testing.T, src string) (*os.File, *Updater) {
	t.Helper()

	file, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}

	st, _ := file.Stat()
	z, err := NewUpdater(file, st.Size())
	if err != nil {
		file.Close()
		t.Fatal(err)
	}

	return file, z
}

func testAddFile(t *testing.T, z *Updater, addfile ZipTestFile) {
	t.Helper()

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
}

func testUpdateFile(t *testing.T, z *Updater, updatefile ZipTestFile) {
	t.Helper()

	wc, err := z.Update(updatefile.Name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wc.Write(updatefile.Content); err != nil {
		t.Fatal(err)
	}
	if err := wc.Close(); err != nil {
		t.Fatal(err)
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
			t.Fatalf("name=%q, want %q", files[i].Name, ztf.Name)
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
