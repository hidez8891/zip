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
	Name       string
	BaseFile   []ZipTestFile
	AppendFile []WriteTest
	ResultFile []ZipTestFile
}

var updateTests = []UpdaterTest{
	{
		Name: "test.zip",
		BaseFile: []ZipTestFile{
			{
				Name:    "test.txt",
				Content: []byte("This is a test text file.\n"),
			},
			{
				Name: "gophercolor16x16.png",
				File: "gophercolor16x16.png",
			},
		},
		AppendFile: []WriteTest{
			{
				Name: "foo",
				Data: []byte("Rabbits, guinea pigs, gophers, marsupial rats, and quolls."),
			},
		},
		ResultFile: []ZipTestFile{
			{
				Name:    "test.txt",
				Content: []byte("This is a test text file.\n"),
			},
			{
				Name: "gophercolor16x16.png",
				File: "gophercolor16x16.png",
			},
			{
				Name:    "foo",
				Content: []byte("Rabbits, guinea pigs, gophers, marsupial rats, and quolls."),
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
	if len(zu.Files()) != len(zt.BaseFile) {
		t.Fatalf("file count=%d, want %d", len(zu.Files()), len(zt.BaseFile))
	}
	for _, ft := range zt.BaseFile {
		updateReadTestFile(t, zu, &ft)
	}

	b := new(bytes.Buffer)
	for _, ft := range zt.AppendFile {
		updateWriteTestFile(t, zu, &ft)
	}
	if err := zu.SaveAs(b); err != nil {
		t.Fatalf("SaveAs error=%v", err)
	}

	zr, err := NewUpdater(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Discard()
	if len(zr.Files()) != len(zt.ResultFile) {
		t.Fatalf("file count=%d, want %d", len(zu.Files()), len(zt.BaseFile))
	}
	for _, ft := range zt.BaseFile {
		updateReadTestFile(t, zu, &ft)
	}
}

func updateReadTestFile(t *testing.T, zu *Updater, ft *ZipTestFile) {
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

func updateWriteTestFile(t *testing.T, zu *Updater, ft *WriteTest) {
	w, err := zu.Create(ft.Name)
	if err != nil {
		t.Fatalf("%s: Create error=%v", ft.Name, err)
	}
	defer w.Close()

	_, err = w.Write(ft.Data)
	if err != nil {
		t.Fatalf("%s: Write error=%v", ft.Name, err)
	}
}
