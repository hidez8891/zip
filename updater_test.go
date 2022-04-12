package zip

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type UpdaterTest struct {
	Name string
	File []ZipTestFile
}

var updateTests = []UpdaterTest{
	{
		Name: "test.zip",
		File: []ZipTestFile{
			{
				Name:    "test.txt",
				Content: []byte("This is a test text file.\n"),
			},
			{
				Name: "gophercolor16x16.png",
				File: "gophercolor16x16.png",
			},
		},
	},
}

func TestUpdater(t *testing.T) {
	for _, zt := range updateTests {
		t.Run(zt.Name, func(t *testing.T) {
			updateTestZip(t, zt)
		})
	}
}

func updateTestZip(t *testing.T, zt UpdaterTest) {
	path := filepath.Join("testdata", zt.Name)
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("os.Stat(%s): %v", zt.Name, err)
		return
	}
	r, err := os.Open(path)
	if err != nil {
		t.Errorf("os.Open(%s): %v", zt.Name, err)
		return
	}
	defer r.Close()

	zu, err := NewUpdater(r, info.Size())
	if err != nil {
		t.Errorf("NewReader(%s): %v", zt.Name, err)
		return
	}
	defer zu.Discard()

	if len(zu.Files()) != len(zt.File) {
		t.Fatalf("file count=%d, want %d", len(zu.Files()), len(zt.File))
	}

	for _, ft := range zt.File {
		updateReadTestFile(t, zu, ft, zt.Name)
	}
}

func updateReadTestFile(t *testing.T, zu *Updater, ft ZipTestFile, srcName string) {
	found := false
	for _, info := range zu.Files() {
		if info.Name() == ft.Name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("%s: %s is not found", srcName, ft.Name)
		return
	}

	r, err := zu.Open(ft.Name)
	if err != nil {
		t.Errorf("%s - %s: Open error=%v", srcName, ft.Name, err)
		return
	}

	var b bytes.Buffer
	_, err = io.Copy(&b, r)
	if err != nil {
		t.Errorf("%s - %s: Read error=%v", srcName, ft.Name, err)
		return
	}
	r.Close()

	var c []byte
	if ft.Content != nil {
		c = ft.Content
	} else if c, err = os.ReadFile("testdata/" + ft.File); err != nil {
		t.Error(err)
		return
	}

	if b.Len() != len(c) {
		t.Errorf("%s - %s: len=%d, want %d", srcName, ft.Name, b.Len(), len(c))
		return
	}

	for i, b := range b.Bytes() {
		if b != c[i] {
			t.Errorf("%s - %s: content[%d]=%q want %q", srcName, ft.Name, i, b, c[i])
			return
		}
	}
}
