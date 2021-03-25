package pipelinetotaskrun

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"testing"
)

func TestPutTasksInOrderUnsupported(t *testing.T) {
	for _, tc := range []struct {
		name  string
		tasks []v1beta1.PipelineTask
	}{{
		name: "parallel",
		tasks: []v1beta1.PipelineTask{{
			Name: "starts",
		}, {
			Name: "alsostarts",
		}},
	}, {
		name: "fan in",
		tasks: []v1beta1.PipelineTask{{
			Name: "out1",
		}, {
			Name: "out2",
		}, {
			Name:     "in",
			RunAfter: []string{"out1", "out2"},
		}},
	}, {
		name: "fan out",
		tasks: []v1beta1.PipelineTask{{
			Name: "first",
		}, {
			Name:     "out1",
			RunAfter: []string{"first"},
		}, {
			Name:     "out2",
			RunAfter: []string{"first"},
		}},
	}, {
		name: "cycle",
		tasks: []v1beta1.PipelineTask{{
			Name:     "first",
			RunAfter: []string{"second"},
		}, {
			Name:     "second",
			RunAfter: []string{"first"},
		}},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := putTasksInOrder(tc.tasks)
			if err == nil {
				t.Fatalf("expected error for unsupported tasks but got none")
			}
		})
	}
}

func TestPutTasksInOrder(t *testing.T) {
	for _, tc := range []struct {
		name          string
		tasks         []v1beta1.PipelineTask
		expectedOrder []string
	}{{
		name: "oneTask",
		tasks: []v1beta1.PipelineTask{{
			Name: "first",
		}},
		expectedOrder: []string{"first"},
	}, {
		name: "twoTask",
		tasks: []v1beta1.PipelineTask{{
			Name: "first",
		}, {
			Name:     "second",
			RunAfter: []string{"first"},
		}},
		expectedOrder: []string{"first", "second"},
	}, {
		name: "twoTaskReversed",
		tasks: []v1beta1.PipelineTask{{
			Name:     "second",
			RunAfter: []string{"first"},
		}, {
			Name: "first",
		}},
		expectedOrder: []string{"first", "second"},
	}, {
		name: "threeTask",
		tasks: []v1beta1.PipelineTask{{
			Name: "first",
		}, {
			Name:     "second",
			RunAfter: []string{"first"},
		}, {
			Name:     "third",
			RunAfter: []string{"second"},
		}},
		expectedOrder: []string{"first", "second", "third"},
	}, {
		name: "threeTaskUnordered1",
		tasks: []v1beta1.PipelineTask{{
			Name:     "second",
			RunAfter: []string{"first"},
		}, {
			Name:     "third",
			RunAfter: []string{"second"},
		}, {
			Name: "first",
		}},
		expectedOrder: []string{"first", "second", "third"},
	}, {
		name: "threeTaskUnordered2",
		tasks: []v1beta1.PipelineTask{{
			Name:     "third",
			RunAfter: []string{"second"},
		}, {
			Name: "first",
		}, {
			Name:     "second",
			RunAfter: []string{"first"},
		}},
		expectedOrder: []string{"first", "second", "third"},
	}, {
		name: "threeTaskReversed",
		tasks: []v1beta1.PipelineTask{{
			Name:     "third",
			RunAfter: []string{"second"},
		}, {
			Name:     "second",
			RunAfter: []string{"first"},
		}, {
			Name: "first",
		}},
		expectedOrder: []string{"first", "second", "third"},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			sequence, err := putTasksInOrder(tc.tasks)
			if err != nil {
				t.Fatalf("did not expect error but got %v", err)
			}
			if len(sequence) != len(tc.expectedOrder) {
				t.Fatalf("returned sequence had length %d but expected %d", len(sequence), len(tc.expectedOrder))
			}
			for i := 0; i < len(sequence); i++ {
				if sequence[i].Name != tc.expectedOrder[i] {
					t.Errorf("expected %s in position %d but got %s", tc.expectedOrder[i], i, sequence[i].Name)
				}
			}

		})
	}
}
