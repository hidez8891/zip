// Copyright 2018 hidez8891. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zip

import (
	"bytes"
	"io"
	"testing"
)

func TestUpdaterOnlyRead(t *testing.T) {
	var testcase []ZipTestFile
	file := "winxp.zip"
	for _, tt := range tests {
		if tt.Name == file {
			testcase = tt.File
			break
		}
	}

	z, err := NewUpdater("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()

	files := z.Files()
	if len(testcase) != len(files) {
		t.Fatalf("file count=%d, want %d", len(testcase), len(files))
	}
	for i, tc := range testcase {
		if tc.Name != files[i].Name {
			t.Errorf("name=%q, want %q", tc.Name, files[i].Name)
		}

		r, err := z.Open(tc.Name)
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
		if len(buf) != len(tc.Content) {
			t.Fatalf("filesize len=%d, want %d", len(buf), len(tc.Content))
		}
		for i, c := range tc.Content {
			if c != buf[i] {
				t.Fatalf("content[%d]=%q, want %q", i, c, buf[i])
			}
		}
	}
}
