package cel

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseFilter(t *testing.T) {
	env, err := NewEnv()
	if err != nil {
		t.Fatalf("NewEnv: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		for _, s := range []string{
			"",
			"taskrun",
			"pipelinerun",
			"result",
			"result.id",
			`result.id == "1"`,
			`result.id == "1" || taskrun.api_version == "2"`,
			`result.id.startsWith("tacocat")`,
		} {
			t.Run(s, func(t *testing.T) {
				if _, err := ParseFilter(env, s); err != nil {
					t.Fatal(err)
				}
			})
		}
	})

	t.Run("error", func(t *testing.T) {
		for _, s := range []string{
			"asdf",
			"result.id == 1", // string != int
			"result.ID",      // case sensitive
		} {
			t.Run(s, func(t *testing.T) {
				if p, err := ParseFilter(env, s); status.Code(err) != codes.InvalidArgument {
					t.Fatalf("expected error, got: (%v, %v)", p, err)
				}
			})
		}
	})
}
