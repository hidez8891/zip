package zip

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type updateInst int

const (
	updaterWriteID updateInst = iota
	updaterAppendID
	updaterRenameID
	updaterDeleteID
)

type updaterCmd struct {
	id updateInst

	writes  []WriteTest
	appends []WriteTest
	renames [][]string
	deletes []string
}

type UpdaterTest struct {
	Name         string
	OriginalFile []ZipTestFile
	Commands     []*updaterCmd
	ResultFile   []ZipTestFile
}

var updateTests = []UpdaterTest{
	{
		Name: "test.zip",
		OriginalFile: []ZipTestFile{
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
		Commands: []*updaterCmd{
			{
				id: updaterWriteID,
				writes: []WriteTest{
					{
						Name: "foo",
						Data: []byte("Rabbits, guinea pigs, gophers, marsupial rats, and quolls."),
					},
				},
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
		Commands: []*updaterCmd{
			{
				id: updaterWriteID,
				writes: []WriteTest{
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
		Commands: []*updaterCmd{
			{
				id: updaterDeleteID,
				deletes: []string{
					"test.txt",
				},
			},
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
		Commands: []*updaterCmd{
			{
				id: updaterRenameID,
				renames: [][]string{
					{"test.txt", "test2.txt"},
				},
			},
		},
		ResultFile: []ZipTestFile{
			{
				Name: "gophercolor16x16.png",
				File: "gophercolor16x16.png",
			},
			{
				// renamed files are moved to the end.
				Name:    "test2.txt",
				Content: []byte("This is a test text file.\n"),
			},
		},
	},
	{
		Name: "test.zip",
		Commands: []*updaterCmd{
			{
				id: updaterAppendID,
				appends: []WriteTest{
					{
						// writing from file to temp
						Name: "test.txt",
						Data: []byte("Hello Golang.\n"),
					},
					{
						// writing from temp to temp
						Name: "test.txt",
						Data: []byte("Hello World.\n"),
					},
				},
			},
		},
		ResultFile: []ZipTestFile{
			{
				Name: "gophercolor16x16.png",
				File: "gophercolor16x16.png",
			},
			{
				// edited files are moved to the end.
				Name:    "test.txt",
				Content: []byte("This is a test text file.\nHello Golang.\nHello World.\n"),
			},
		},
	},
}

func TestUpdater(t *testing.T) {
	for _, zt := range updateTests {
		t.Run(zt.Name, func(t *testing.T) {
			testUpdateZip(t, zt)
		})
	}
}

func testUpdateZip(t *testing.T, zt UpdaterTest) {
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

	if zt.OriginalFile != nil {
		if len(zu.Files()) != len(zt.OriginalFile) {
			t.Fatalf("file count=%d, want %d", len(zu.Files()), len(zt.OriginalFile))
		}
		for i, ft := range zt.OriginalFile {
			testUpdateReadFile(t, zu, i, &ft)
		}
	}

	if zt.Commands != nil {
		for _, cmd := range zt.Commands {
			if cmd.id == updaterWriteID {
				for _, ft := range cmd.writes {
					testUpdateWriteFile(t, zu, &ft)
				}
			}
			if cmd.id == updaterAppendID {
				for _, ft := range cmd.appends {
					testUpdateAppendFile(t, zu, &ft)
				}
			}
			if cmd.id == updaterRenameID {
				for _, rn := range cmd.renames {
					if err := zu.Rename(rn[0], rn[1]); err != nil {
						t.Fatalf("Rename error=%v", err)
					}
				}
			}
			if cmd.id == updaterDeleteID {
				for _, name := range cmd.deletes {
					if err := zu.Delete(name); err != nil {
						t.Fatalf("Delete error=%v", err)
					}
				}
			}
		}

		b := new(bytes.Buffer)
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
		for i, ft := range zt.ResultFile {
			testUpdateReadFile(t, zr, i, &ft)
		}
	}
}

func testUpdateReadFile(t *testing.T, zu *Updater, index int, ft *ZipTestFile) {
	file := zu.Files()[index]
	if file.Name() != ft.Name {
		t.Fatalf("file name %q, want %q", file.Name(), ft.Name)
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

func testUpdateWriteFile(t *testing.T, zu *Updater, ft *WriteTest) {
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

func testUpdateAppendFile(t *testing.T, zu *Updater, ft *WriteTest) {
	r, w, err := zu.Update(ft.Name)
	if err != nil {
		t.Fatalf("%s: Update error=%v", ft.Name, err)
	}
	defer r.Close()
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		t.Fatalf("%s: io.Copy error=%v", ft.Name, err)
	}
	_, err = w.Write(ft.Data)
	if err != nil {
		t.Fatalf("%s: Write error=%v", ft.Name, err)
	}
}
