package writer

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriteTrigger(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := WriteTrigger("./testdata/spec-full.yaml", buf); err != nil {
		t.Fatalf("error from 'WriteTrigger': %v", err)
	}
	got := buf.Bytes()

	path := "./testdata/config.yaml"
	want, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("WriteTrigger mismatch (-want +got):\n %s", diff)
	}
}

func TestWritePipelineRun(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := WritePipelineRun("./testdata/spec-full.yaml", buf); err != nil {
		t.Fatalf("error from 'WritePipelineRun': %v", err)
	}
	got := buf.Bytes()

	path := "./testdata/run.yaml"
	want, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("WritePipelineRun mismatch (-want +got):\n %s", diff)
	}
}
