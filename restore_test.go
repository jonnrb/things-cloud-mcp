package main

import (
	"context"
	"testing"
)

func TestRestoringFromTheTrash(t *testing.T) {
	fc := newFakeCloud("test@example.com", makeTaskItem("task-1", withTitle("Call"), withTrashed()))
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	assertRefused(t, fc, func() (string, bool) {
		r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "task-1", "title": "Call back"}))
		if err != nil {
			t.Fatal(err)
		}
		return resultText(t, r), r.IsError
	}, "task not found")

	r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "task-1", "status": "restored"}))
	if err != nil {
		t.Fatal(err)
	}
	assertNotError(t, r)
	if written := lastCommit(t, fc)["task-1"]; written["tr"] != false {
		t.Fatalf("wrote %v, want the to-do out of the Trash", written)
	}
	if tmcp.state.Tasks["task-1"].InTrash {
		t.Fatal("to-do still in the Trash after restoring")
	}
}
