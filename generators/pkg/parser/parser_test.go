package parser

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	file, err := os.Open("testdata/products.yaml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(file)
	if err != nil {
		t.Fatalf("error from 'Parse': %v", err)
	}

	want := products{[]int{1, 2, 3}, []string{"one", "two", "three"}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Products mismatch (-want +got):\n %s", diff)
	}

	file.Close()
}
