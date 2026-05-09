package agent

import "testing"

func TestDecomposeTasks_OneRequestOneTask(t *testing.T) {
	for _, msg := range []string{
		"hello",
		"a;b;c",
		"line1\nline2\nline3",
	} {
		got := decomposeTasks(msg)
		if len(got) != 1 || got[0] != msg {
			t.Fatalf("msg %q got %+v", msg, got)
		}
	}
}
