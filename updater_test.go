package zip

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/exp/slices"
)

type UpdaterTest struct {
	Name       string
	BaseFile   []ZipTestFile
	AppendFile []WriteTest
	RenameFile [][]string
	DeleteFile []string
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
	},
	{
		Name: "test.zip",
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
	{
		Name: "test.zip",
		AppendFile: []WriteTest{
			{
				Name: "foo",
				Data: []byte("Rabbits."),
			},
			{
				Name: "foo",
				Data: []byte("Gophers."), // overwrite
			},
			{
				Name: "test.txt",
				Data: []byte("This is a overwrite text file.\n"), // overwrite
			},
		},
		ResultFile: []ZipTestFile{
			{
				Name: "gophercolor16x16.png",
				File: "gophercolor16x16.png",
			},
			{
				Name:    "foo",
				Content: []byte("Gophers."), // overwrite
			},
			{
				Name:    "test.txt",
				Content: []byte("This is a overwrite text file.\n"), // overwrite
			},
		},
	},
	{
		Name: "test.zip",
		DeleteFile: []string{
			"test.txt",
		},
		ResultFile: []ZipTestFile{
			{
				Name: "gophercolor16x16.png",
				File: "gophercolor16x16.png",
			},
		},
	},
	{
		Name: "test.zip",
		RenameFile: [][]string{
			{"test.txt", "test2.txt"},
		},
		ResultFile: []ZipTestFile{
			{
				Name:    "test2.txt",
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

func TestUpdaterUpdateFile(t *testing.T) {
	inbuf := new(bytes.Buffer)
	func() {
		zw := NewWriter(inbuf)
		w, err := zw.Create("test.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("Hello")); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	outbuf := new(bytes.Buffer)
	func() {
		zu, err := NewUpdater(bytes.NewReader(inbuf.Bytes()), int64(inbuf.Len()))
		if err != nil {
			t.Fatal(err)
		}
		defer zu.Discard()

		r, w, err := zu.Update("test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		if _, err := io.Copy(w, r); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(" World")); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if err := zu.SaveAs(outbuf); err != nil {
			t.Fatal(err)
		}
	}()

	func() {
		expect := "Hello World"

		zr, err := NewReader(bytes.NewReader(outbuf.Bytes()), int64(outbuf.Len()))
		if err != nil {
			t.Fatal(err)
		}
		r, err := zr.Open("test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r); err != nil {
			t.Fatal(err)
		}
		if buf.Len() != len(expect) {
			t.Fatalf("file size=%d, want %d", buf.Len(), len(expect))
		}
		for i, b := range buf.Bytes() {
			if b != expect[i] {
				t.Errorf("file content[%d]=%q want %q", i, b, expect[i])
				return
			}
		}
	}()
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

	if zt.BaseFile != nil {
		if len(zu.Files()) != len(zt.BaseFile) {
			t.Fatalf("file count=%d, want %d", len(zu.Files()), len(zt.BaseFile))
		}
		for _, ft := range zt.BaseFile {
			updateReadTestFile(t, zu, &ft)
		}
	}

	if zt.ResultFile != nil {
		b := new(bytes.Buffer)

		if zt.AppendFile != nil {
			for _, ft := range zt.AppendFile {
				updateWriteTestFile(t, zu, &ft)
			}
		}
		if zt.RenameFile != nil {
			for _, rn := range zt.RenameFile {
				if err := zu.Rename(rn[0], rn[1]); err != nil {
					t.Fatalf("Rename error=%v", err)
				}
			}
		}
		if zt.DeleteFile != nil {
			for _, name := range zt.DeleteFile {
				if err := zu.Delete(name); err != nil {
					t.Fatalf("Delete error=%v", err)
				}
			}
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
			t.Fatalf("file count=%d, want %d", len(zr.Files()), len(zt.ResultFile))
		}
		for _, ft := range zt.ResultFile {
			updateReadTestFile(t, zr, &ft)
		}
	}
}

func updateReadTestFile(t *testing.T, zu *Updater, ft *ZipTestFile) {
	files := zu.Files()
	index := slices.IndexFunc(files, func(f fs.FileInfo) bool {
		return f.Name() == ft.Name
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
