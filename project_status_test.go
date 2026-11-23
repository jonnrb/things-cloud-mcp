package main

import (
	"context"
	"testing"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
)

func TestFinishingAProjectWithOpenToDosIsRefused(t *testing.T) {
	for _, status := range []string{"completed", "canceled"} {
		t.Run(status, func(t *testing.T) {
			fc := newFakeCloud("test@example.com",
				makeTaskItem("project-1", withTitle("Trip"), withTaskType(thingscloud.TaskTypeProject)),
				makeTaskItem("heading-1", withTitle("Packing"), withTaskType(thingscloud.TaskTypeHeading), withParent("project-1")),
				makeTaskItem("task-1", withTitle("Book flights"), withParent("project-1")),
				makeTaskItem("task-2", withTitle("Pack"), withActionGroup("heading-1")),
				makeTaskItem("task-3", withTitle("Done already"), withParent("project-1"), withStatus(thingscloud.TaskStatusCompleted)),
				makeTaskItem("task-4", withTitle("Trashed"), withParent("project-1"), withTrashed()),
			)
			defer fc.Close()
			tmcp := newTestThingsMCP(t, fc)

			assertRefused(t, fc, func() (string, bool) {
				r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "project-1", "status": status}))
				if err != nil {
					t.Fatal(err)
				}
				return resultText(t, r), r.IsError
			}, "still has 2 open to-do(s)")
		})
	}
}

func TestFinishingAProjectWithNothingOpen(t *testing.T) {
	fc := newFakeCloud("test@example.com",
		makeTaskItem("project-1", withTitle("Trip"), withTaskType(thingscloud.TaskTypeProject)),
		makeTaskItem("task-1", withTitle("Book flights"), withParent("project-1"), withStatus(thingscloud.TaskStatusCompleted)),
		makeTaskItem("task-2", withTitle("Hotel"), withParent("project-1"), withStatus(thingscloud.TaskStatusCanceled)),
		makeTaskItem("template", withTitle("Check in"), withParent("project-1"), withRepeat(0)),
	)
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "project-1", "status": "completed"}))
	if err != nil {
		t.Fatal(err)
	}
	assertNotError(t, r)
	if written := lastCommit(t, fc)["project-1"]; written["ss"] != float64(thingscloud.TaskStatusCompleted) {
		t.Fatalf("wrote %v, want the project completed", written)
	}
}

func TestTrashingAProjectWithOpenToDos(t *testing.T) {
	fc := newFakeCloud("test@example.com",
		makeTaskItem("project-1", withTitle("Trip"), withTaskType(thingscloud.TaskTypeProject)),
		makeTaskItem("task-1", withTitle("Book flights"), withParent("project-1")),
	)
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "project-1", "status": "trashed"}))
	if err != nil {
		t.Fatal(err)
	}
	assertNotError(t, r)
}
