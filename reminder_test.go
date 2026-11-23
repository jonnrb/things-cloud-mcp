package main

import (
	"context"
	"testing"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
)

func withReminder(secs int) taskOption {
	return func(p *thingscloud.TaskActionItemPayload) {
		p.AlarmTimeOffset = &secs
	}
}

func TestCreatingWithAReminder(t *testing.T) {
	fc := newFakeCloud("test@example.com")
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	for _, handle := range []func(context.Context, map[string]any) (string, map[string]any){
		func(ctx context.Context, args map[string]any) (string, map[string]any) {
			args["title"] = "Call the dentist"
			r, err := tmcp.handleCreateTask(ctx, makeReq(args))
			if err != nil {
				t.Fatal(err)
			}
			assertNotError(t, r)
			uuid := resultJSON[map[string]string](t, r)["uuid"]
			return uuid, lastCommit(t, fc)[uuid]
		},
		func(ctx context.Context, args map[string]any) (string, map[string]any) {
			args["title"] = "Plan the trip"
			r, err := tmcp.handleCreateProject(ctx, makeReq(args))
			if err != nil {
				t.Fatal(err)
			}
			assertNotError(t, r)
			uuid := resultJSON[map[string]string](t, r)["uuid"]
			return uuid, lastCommit(t, fc)[uuid]
		},
	} {
		_, written := handle(context.Background(), map[string]any{"schedule": "2026-11-25", "reminder_time": "14:30"})
		if written["ato"] != float64(14*3600+30*60) {
			t.Fatalf("ato = %v, want 14:30 as seconds after midnight", written["ato"])
		}
		if written["sr"] != float64(1795564800) {
			t.Fatalf("sr = %v, want Nov 25", written["sr"])
		}
		if v, ok := written["rmd"]; !ok || v != nil {
			t.Fatalf("rmd = %v (present %v), want null", v, ok)
		}
	}
}

func TestCreatingAReminderWithoutADayIsRefused(t *testing.T) {
	for name, args := range map[string]map[string]any{
		"no schedule":       {"title": "Call", "reminder_time": "14:30"},
		"anytime":           {"title": "Call", "schedule": "anytime", "reminder_time": "14:30"},
		"someday":           {"title": "Call", "schedule": "someday", "reminder_time": "14:30"},
		"in a project only": {"title": "Call", "project_uuid": "project-1", "reminder_time": "14:30"},
	} {
		t.Run(name, func(t *testing.T) {
			fc := newFakeCloud("test@example.com", makeTaskItem("project-1", withTitle("Project"), withTaskType(thingscloud.TaskTypeProject)))
			defer fc.Close()
			tmcp := newTestThingsMCP(t, fc)
			assertRefused(t, fc, func() (string, bool) {
				r, err := tmcp.handleCreateTask(context.Background(), makeReq(args))
				if err != nil {
					t.Fatal(err)
				}
				return resultText(t, r), r.IsError
			}, "a reminder needs a day")
		})
	}
}

func TestReminderDateIsRefused(t *testing.T) {
	fc := newFakeCloud("test@example.com", makeTaskItem("task-1", withTitle("Call")))
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	for name, call := range map[string]func() (string, bool){
		"create task": func() (string, bool) {
			r, err := tmcp.handleCreateTask(context.Background(), makeReq(map[string]any{"title": "Call", "schedule": "today", "reminder_date": "2026-11-25", "reminder_time": "14:30"}))
			if err != nil {
				t.Fatal(err)
			}
			return resultText(t, r), r.IsError
		},
		"create project": func() (string, bool) {
			r, err := tmcp.handleCreateProject(context.Background(), makeReq(map[string]any{"title": "Trip", "reminder_date": "2026-11-25", "reminder_time": "14:30"}))
			if err != nil {
				t.Fatal(err)
			}
			return resultText(t, r), r.IsError
		},
		"edit item": func() (string, bool) {
			r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "task-1", "reminder_date": "none"}))
			if err != nil {
				t.Fatal(err)
			}
			return resultText(t, r), r.IsError
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertRefused(t, fc, call, "reminder_date isn't taken")
		})
	}
}

func TestEditingAReminder(t *testing.T) {
	nov25 := float64(1795564800)
	tests := []struct {
		name    string
		task    []taskOption
		args    map[string]any
		want    map[string]any // fields written, nil for null
		absent  []string       // fields not written
		refused string
	}{
		{
			name:   "set on a dated to-do",
			task:   []taskOption{withSchedule(thingscloud.TaskScheduleSomeday), withScheduledDate(nov22.AddDate(0, 0, 3))},
			args:   map[string]any{"reminder_time": "09:00"},
			want:   map[string]any{"ato": float64(9 * 3600)},
			absent: []string{"rmd", "sr", "st"},
		},
		{
			name:    "set on an undated to-do",
			task:    []taskOption{withSchedule(thingscloud.TaskScheduleAnytime)},
			args:    map[string]any{"reminder_time": "09:00"},
			refused: "a reminder needs a day",
		},
		{
			name:    "set while moving to anytime",
			task:    []taskOption{withSchedule(thingscloud.TaskScheduleSomeday), withScheduledDate(nov22.AddDate(0, 0, 3))},
			args:    map[string]any{"schedule": "anytime", "reminder_time": "09:00"},
			refused: "a reminder needs a day",
		},
		{
			name:   "set with a new date",
			task:   []taskOption{withSchedule(thingscloud.TaskScheduleAnytime)},
			args:   map[string]any{"schedule": "2026-11-25", "reminder_time": "09:00"},
			want:   map[string]any{"ato": float64(9 * 3600), "sr": nov25},
			absent: []string{"rmd"},
		},
		{
			name:   "removed",
			task:   []taskOption{withSchedule(thingscloud.TaskScheduleSomeday), withScheduledDate(nov22.AddDate(0, 0, 3)), withReminder(9 * 3600)},
			args:   map[string]any{"reminder_time": "none"},
			want:   map[string]any{"ato": nil},
			absent: []string{"rmd", "sr"},
		},
		{
			name:   "kept when the to-do moves to another day",
			task:   []taskOption{withSchedule(thingscloud.TaskScheduleSomeday), withScheduledDate(nov22.AddDate(0, 0, 3)), withReminder(9 * 3600)},
			args:   map[string]any{"schedule": "2026-11-26"},
			want:   map[string]any{"sr": nov25 + 86400},
			absent: []string{"ato", "rmd"},
		},
		{
			name:   "removed when the to-do loses its day",
			task:   []taskOption{withSchedule(thingscloud.TaskScheduleSomeday), withScheduledDate(nov22.AddDate(0, 0, 3)), withReminder(9 * 3600)},
			args:   map[string]any{"schedule": "someday"},
			want:   map[string]any{"ato": nil, "sr": nil},
			absent: []string{"rmd"},
		},
		{
			name:   "nothing to remove when an undated to-do moves",
			task:   []taskOption{withSchedule(thingscloud.TaskScheduleAnytime)},
			args:   map[string]any{"schedule": "someday"},
			absent: []string{"ato", "rmd"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := newFakeCloud("test@example.com", makeTaskItem("task-1", append([]taskOption{withTitle("Call")}, tt.task...)...))
			defer fc.Close()
			tmcp := newTestThingsMCP(t, fc)

			args := map[string]any{"uuid": "task-1"}
			for k, v := range tt.args {
				args[k] = v
			}
			call := func() (string, bool) {
				r, err := tmcp.handleEditTask(context.Background(), makeReq(args))
				if err != nil {
					t.Fatal(err)
				}
				return resultText(t, r), r.IsError
			}
			if tt.refused != "" {
				assertRefused(t, fc, call, tt.refused)
				return
			}
			if text, isError := call(); isError {
				t.Fatalf("edit refused: %s", text)
			}
			written := lastCommit(t, fc)["task-1"]
			for k, want := range tt.want {
				if got, ok := written[k]; !ok || got != want {
					t.Errorf("%s = %v (written %v), want %v", k, got, ok, want)
				}
			}
			for _, k := range tt.absent {
				if v, ok := written[k]; ok {
					t.Errorf("%s written as %v", k, v)
				}
			}
		})
	}
}
