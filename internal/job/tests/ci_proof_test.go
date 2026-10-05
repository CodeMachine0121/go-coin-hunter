package job_test

import "testing"

func TestCiMustFailOnThisDeliberateFailure(t *testing.T) {
	t.Fatal("deliberate failure to prove CI turns red")
}
