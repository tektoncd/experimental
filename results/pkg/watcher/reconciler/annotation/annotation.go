package annotation

import (
	"encoding/json"

	"gomodules.xyz/jsonpatch/v2"
)

const (
	path     = "/metadata/annotations/results.tekton.dev~1id"
	ResultID = "results.tekton.dev/id"
)

// AddResultID creates a jsonpatch path used for adding results_id to Result
// annotations field.
func AddResultID(resultID string) ([]byte, error) {
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      path,
		Value:     resultID,
	}}
	return json.Marshal(patches)
}
