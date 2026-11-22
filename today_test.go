package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
)

// atNewYorkEvening sets the server's zone to New York and its clock to 8:30 PM
// there on 2026-11-21, when it's already 2026-11-22 in UTC, for one test.
func atNewYorkEvening(t *testing.T) {
	t.Helper()
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	local, oldClock := time.Local, clock
	t.Cleanup(func() { time.Local, clock = local, oldClock })
	time.Local = zone
	clock = func() time.Time { return time.Date(2026, 11, 22, 1, 30, 0, 0, time.UTC).Local() }
}

// lastCommit returns the payloads of the fake cloud's last commit, by UUID.
func lastCommit(t *testing.T, fc *fakeCloud) map[string]map[string]any {
	t.Helper()
	commits := fc.getCommitLog()
	if len(commits) == 0 {
		t.Fatal("nothing was committed")
	}
	var body map[string]struct {
		P map[string]any `json:"p"`
	}
	if err := json.Unmarshal(commits[len(commits)-1], &body); err != nil {
		t.Fatalf("decode commit: %v", err)
	}
	payloads := make(map[string]map[string]any, len(body))
	for id, envelope := range body {
		payloads[id] = envelope.P
	}
	return payloads
}

var (
	nov21 = time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	nov22 = time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
)

func TestTodayIsTheDayInTheServersZone(t *testing.T) {
	atNewYorkEvening(t)
	if got := today(); !got.Equal(nov21) {
		t.Fatalf("today() = %v at 8:30 PM in New York, want %v", got, nov21)
	}
	time.Local = time.UTC
	if got := today(); !got.Equal(nov22) {
		t.Fatalf("today() = %v with the zone UTC, want %v", got, nov22)
	}
}

func TestScheduleTodayInTheEveningIsToday(t *testing.T) {
	atNewYorkEvening(t)
	fc := newFakeCloud("test@example.com", makeTaskItem("task-1", withTitle("Existing")))
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	for _, schedule := range []string{"today", "tonight"} {
		result, err := tmcp.handleCreateTask(context.Background(), makeReq(map[string]any{"title": "Take out the recycling", "schedule": schedule}))
		if err != nil {
			t.Fatal(err)
		}
		assertNotError(t, result)
		created := resultJSON[map[string]string](t, result)["uuid"]
		p := lastCommit(t, fc)[created]
		if p["sr"] != float64(nov21.Unix()) || p["tir"] != float64(nov21.Unix()) {
			t.Fatalf("%s: sr, tir = %v, %v; want %d (2026-11-21)", schedule, p["sr"], p["tir"], nov21.Unix())
		}

		result, err = tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "task-1", "schedule": schedule}))
		if err != nil {
			t.Fatal(err)
		}
		assertNotError(t, result)
		if p := lastCommit(t, fc)["task-1"]; p["sr"] != float64(nov21.Unix()) {
			t.Fatalf("edit to %s: sr = %v, want %d (2026-11-21)", schedule, p["sr"], nov21.Unix())
		}
	}
}

func TestFindTasksTodayInTheEvening(t *testing.T) {
	atNewYorkEvening(t)
	fc := newFakeCloud("test@example.com",
		makeTaskItem("today", withTitle("Due today"), withScheduledDate(nov21)),
		makeTaskItem("tomorrow", withTitle("Due tomorrow"), withSchedule(thingscloud.TaskScheduleSomeday), withScheduledDate(nov22)),
	)
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	result, err := tmcp.handleFindTasks(context.Background(), makeReq(map[string]any{"schedule": "today"}))
	if err != nil {
		t.Fatal(err)
	}
	assertNotError(t, result)
	tasks := resultJSON[[]TaskOutput](t, result)
	if len(tasks) != 1 || tasks[0].UUID != "today" {
		t.Fatalf("today at 8:30 PM in New York = %#v, want only the task for 2026-11-21", tasks)
	}
}

func TestWriteDatesAreCalendarDays(t *testing.T) {
	for in, want := range map[string]time.Time{
		"2026-11-21":                nov21,
		"2026-11-21T23:30:00-05:00": nov21,
		"2026-11-21T00:30:00+09:00": nov21,
	} {
		got := parseDay(in)
		if got == nil || !got.Equal(want) {
			t.Errorf("parseDay(%q) = %v, want %v", in, got, want)
		}
	}
	for _, in := range []string{"", "tomorrow", "2026-13-01", "11/21/2026"} {
		if got := parseDay(in); got != nil {
			t.Errorf("parseDay(%q) = %v, want nil", in, got)
		}
	}

	fc := newFakeCloud("test@example.com", makeTaskItem("task-1", withTitle("Existing")))
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)
	result, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "task-1", "deadline": "2026-11-21T23:30:00-05:00"}))
	if err != nil {
		t.Fatal(err)
	}
	assertNotError(t, result)
	if p := lastCommit(t, fc)["task-1"]; p["dd"] != float64(nov21.Unix()) {
		t.Fatalf("dd = %v, want midnight UTC on 2026-11-21 (%d)", p["dd"], nov21.Unix())
	}
}
