package writer

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriteToDisk(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := WriteToDisk("./testdata/spec-full.yaml", buf); err != nil {
		t.Fatalf("error from 'WriteToDisk': %v", err)
	}
	got := buf.Bytes()

	path := "./testdata/pipeline.yaml"
	want, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("WriteToDisk mismatch (-want +got):\n %s", diff)
	}
}
