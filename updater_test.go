package zip

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
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
		updateReadTestFile(t, zu, ft)
	}
}

func updateReadTestFile(t *testing.T, zu *Updater, ft ZipTestFile) {
	files := zu.Files()
	index := sort.Search(len(files), func(i int) bool {
		return files[i].Name() == ft.Name
	})
	if index == -1 {
		t.Errorf("%s is not found", ft.Name)
		return
	}

	r, err := zu.Open(ft.Name)
	if err != nil {
		t.Errorf("%s: Open error=%v", ft.Name, err)
		return
	}

	var b bytes.Buffer
	_, err = io.Copy(&b, r)
	if err != nil {
		t.Errorf("%s: Read error=%v", ft.Name, err)
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
		t.Errorf("%s: len=%d, want %d", ft.Name, b.Len(), len(c))
		return
	}
	for i, b := range b.Bytes() {
		if b != c[i] {
			t.Errorf("%s: content[%d]=%q want %q", ft.Name, i, b, c[i])
			return
		}
	}
}
