package main

import (
	"context"
	"testing"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
)

func TestMovingAToDoKeepsItsSchedule(t *testing.T) {
	atNewYorkEvening(t)
	fc := newFakeCloud("test@example.com",
		makeAreaItem("area-1", "Home"),
		makeTaskItem("project-1", withTitle("Garage"), withTaskType(thingscloud.TaskTypeProject)),
		makeTaskItem("heading-1", withTitle("Shelves"), withTaskType(thingscloud.TaskTypeHeading), withParent("project-1")),
		makeTaskItem("today", withTitle("In Today"), withScheduledDate(nov21), withTodayIndexRefDate(nov21)),
		makeTaskItem("upcoming", withTitle("On a date"), withSchedule(thingscloud.TaskScheduleSomeday), withScheduledDate(nov22), withTodayIndexRefDate(nov22)),
		makeTaskItem("someday", withTitle("Someday"), withSchedule(thingscloud.TaskScheduleSomeday)),
		makeTaskItem("inbox", withTitle("In the Inbox"), withSchedule(thingscloud.TaskScheduleInbox)),
	)
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	for _, destination := range []map[string]any{
		{"area_uuid": "area-1"},
		{"project_uuid": "project-1"},
		{"heading_uuid": "heading-1"},
	} {
		for _, id := range []string{"today", "upcoming", "someday"} {
			args := map[string]any{"uuid": id}
			for k, v := range destination {
				args[k] = v
			}
			result, err := tmcp.handleEditTask(context.Background(), makeReq(args))
			if err != nil {
				t.Fatal(err)
			}
			assertNotError(t, result)
			p := lastCommit(t, fc)[id]
			for _, key := range []string{"st", "sr", "tir"} {
				if value, ok := p[key]; ok {
					t.Errorf("moving %s to %v wrote %s=%v; its schedule should stay", id, destination, key, value)
				}
			}
		}
	}

	result, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "inbox", "area_uuid": "area-1"}))
	if err != nil {
		t.Fatal(err)
	}
	assertNotError(t, result)
	p := lastCommit(t, fc)["inbox"]
	if p["st"] != float64(thingscloud.TaskScheduleAnytime) || p["sr"] != nil || p["tir"] != nil {
		t.Fatalf("moving an Inbox to-do to an area wrote st, sr, tir = %v, %v, %v; want Anytime with no date", p["st"], p["sr"], p["tir"])
	}
	if _, ok := p["sr"]; !ok {
		t.Fatal("moving an Inbox to-do to an area left out sr; st, sr and tir go together")
	}

	// An explicit schedule still wins.
	result, err = tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "today", "area_uuid": "area-1", "schedule": "someday"}))
	if err != nil {
		t.Fatal(err)
	}
	assertNotError(t, result)
	if p := lastCommit(t, fc)["today"]; p["st"] != float64(thingscloud.TaskScheduleSomeday) {
		t.Fatalf("st = %v, want Someday", p["st"])
	}
}
