package main

import (
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
)

// withRepeat makes a task a repeat's template. kind is the rule's tp: 0
// repeats on a schedule, 1 after completion.
func withRepeat(kind int) taskOption {
	return func(p *thingscloud.TaskActionItemPayload) {
		p.Repeater = &thingscloud.RepeaterConfiguration{
			FrequencyUnit:       thingscloud.FrequencyUnitDaily,
			FrequencyAmplitude:  1,
			DetailConfiguration: []thingscloud.RepeaterDetailConfiguration{{}},
			Version:             4,
			Type:                kind,
		}
	}
}

// withTemplate makes a task an instance of the repeat whose template is uuid.
func withTemplate(uuid string) taskOption {
	return func(p *thingscloud.TaskActionItemPayload) {
		ids := []string{uuid}
		p.RecurrenceTaskIDs = &ids
	}
}

// withNewerRepeatRule gives an item a non-null rp, which the SDK's payload
// type doesn't carry.
func withNewerRepeatRule(t *testing.T, item thingscloud.Item) thingscloud.Item {
	t.Helper()
	var p map[string]any
	if err := json.Unmarshal(item.P, &p); err != nil {
		t.Fatal(err)
	}
	p["rp"] = "opaque"
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	item.P = raw
	return item
}

func assertRefused(t *testing.T, fc *fakeCloud, call func() (string, bool), want string) {
	t.Helper()
	text, isError := call()
	if !isError {
		t.Fatalf("write was accepted: %s", text)
	}
	if !strings.Contains(text, want) {
		t.Fatalf("error %q does not contain %q", text, want)
	}
	if commits := fc.getCommitLog(); len(commits) != 0 {
		t.Fatalf("refused write posted %d commits", len(commits))
	}
}

func TestRepeatRulesAreNotWritten(t *testing.T) {
	fc := newFakeCloud("test@example.com", makeTaskItem("task-1", withTitle("Plain")))
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	for name, call := range map[string]func() (string, bool){
		"create task": func() (string, bool) {
			r, err := tmcp.handleCreateTask(context.Background(), makeReq(map[string]any{"title": "Water plants", "recurrence": "daily"}))
			if err != nil {
				t.Fatal(err)
			}
			return resultText(t, r), r.IsError
		},
		"create project": func() (string, bool) {
			r, err := tmcp.handleCreateProject(context.Background(), makeReq(map[string]any{"title": "Monthly review", "recurrence": "monthly"}))
			if err != nil {
				t.Fatal(err)
			}
			return resultText(t, r), r.IsError
		},
		"edit item": func() (string, bool) {
			r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "task-1", "recurrence": "weekly"}))
			if err != nil {
				t.Fatal(err)
			}
			return resultText(t, r), r.IsError
		},
		"clear a repeat": func() (string, bool) {
			r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "task-1", "recurrence": "none"}))
			if err != nil {
				t.Fatal(err)
			}
			return resultText(t, r), r.IsError
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertRefused(t, fc, call, "set the repeat in the Things app")
		})
	}
}

func TestStatusOfARepeatingToDo(t *testing.T) {
	tests := []struct {
		name     string
		template func(t *testing.T) []thingscloud.Item
		// refused is the error a status change gets, by status; the others
		// are written, with the template too when writesTemplate says so.
		refused        map[string]string
		writesTemplate bool
	}{
		{
			name: "repeats on a schedule",
			template: func(*testing.T) []thingscloud.Item {
				return []thingscloud.Item{makeTaskItem("template", withTitle("Water plants"), withRepeat(0))}
			},
		},
		{
			name: "repeats after completion",
			template: func(*testing.T) []thingscloud.Item {
				return []thingscloud.Item{makeTaskItem("template", withTitle("Water plants"), withRepeat(1))}
			},
			refused:        map[string]string{"trashed": "trashing or restoring it"},
			writesTemplate: true,
		},
		{
			name: "template missing",
			template: func(*testing.T) []thingscloud.Item {
				return nil
			},
			refused: map[string]string{"completed": "can't be read here", "canceled": "can't be read here", "trashed": "can't be read here"},
		},
		{
			name: "template has a newer repeat rule",
			template: func(t *testing.T) []thingscloud.Item {
				return []thingscloud.Item{withNewerRepeatRule(t, makeTaskItem("template", withTitle("Water plants"), withRepeat(0)))}
			},
			refused: map[string]string{"completed": "can't be read here", "canceled": "can't be read here", "trashed": "can't be read here"},
		},
	}
	for _, tt := range tests {
		for _, status := range []string{"completed", "canceled", "trashed"} {
			t.Run(tt.name+"/"+status, func(t *testing.T) {
				items := append(tt.template(t), makeTaskItem("instance", withTitle("Water plants"), withTemplate("template")))
				fc := newFakeCloud("test@example.com", items...)
				defer fc.Close()
				tmcp := newTestThingsMCP(t, fc)

				call := func() (string, bool) {
					r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "instance", "status": status}))
					if err != nil {
						t.Fatal(err)
					}
					return resultText(t, r), r.IsError
				}
				if want := tt.refused[status]; want != "" {
					assertRefused(t, fc, call, want)
					return
				}
				if text, isError := call(); isError {
					t.Fatalf("status change refused: %s", text)
				}
				written := lastCommit(t, fc)
				want := []string{"instance"}
				if tt.writesTemplate {
					want = append(want, "template")
				}
				if got := slices.Sorted(maps.Keys(written)); !slices.Equal(got, want) {
					t.Fatalf("wrote %v, want %v", written, want)
				}
			})
		}
	}
}

// afterCompletion is a rule that repeats every n units after completion.
func afterCompletion(unit thingscloud.FrequencyUnit, n int64) thingscloud.RepeaterConfiguration {
	return thingscloud.RepeaterConfiguration{
		FrequencyUnit:       unit,
		FrequencyAmplitude:  n,
		DetailConfiguration: []thingscloud.RepeaterDetailConfiguration{},
		Version:             4,
		Type:                1,
	}
}

// withRule makes a task a repeat's template with the given rule.
func withRule(rule thingscloud.RepeaterConfiguration) taskOption {
	return func(p *thingscloud.TaskActionItemPayload) {
		p.Repeater = &rule
	}
}

// inNewYorkAt sets the server's zone to New York and its clock to the given
// local time there, for one test.
func inNewYorkAt(t *testing.T, year int, month time.Month, day, hour, minute int) {
	t.Helper()
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	local, oldClock := time.Local, clock
	t.Cleanup(func() { time.Local, clock = local, oldClock })
	time.Local = zone
	clock = func() time.Time { return time.Date(year, month, day, hour, minute, 0, 0, zone) }
}

func day(year int, month time.Month, d int) time.Time {
	return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
}

// The next to-do of a repeat after completion starts the rule's interval
// after the day the last one was done, less the days it starts before its
// deadline. A real account's Logbook bears out the first three: a daily repeat
// done at 9:39 PM came back the next day; one every 3 months with deadlines,
// starting a week early, done on 8-07 came back on 10-31 with a deadline of
// 11-07; and one every 4 weeks with deadlines, starting 2 days early, with a
// Sunday left over from a weekly schedule, done on 10-30 came back on 11-25
// with a deadline of 11-27.
func TestCompletingARepeatAfterCompletionSchedulesTheNextOne(t *testing.T) {
	sunday := time.Sunday
	never := thingscloud.Timestamp(time.Unix(8751978978, 871346000))
	tests := []struct {
		name          string
		rule          thingscloud.RepeaterConfiguration
		withDeadlines bool
		done          time.Time // local time the to-do is completed
		wantDone      time.Time
		wantNext      time.Time
	}{
		{
			name:     "every day, in the evening",
			rule:     afterCompletion(thingscloud.FrequencyUnitDaily, 1),
			done:     time.Date(2026, 11, 12, 21, 39, 0, 0, time.UTC),
			wantDone: day(2026, 11, 12),
			wantNext: day(2026, 11, 13),
		},
		{
			name: "every 3 months, starting a week before the deadline",
			rule: func() thingscloud.RepeaterConfiguration {
				r := afterCompletion(thingscloud.FrequencyUnitMonthly, 3)
				r.TimeShift = -7
				r.LastScheduledAt = &never
				return r
			}(),
			withDeadlines: true,
			done:          time.Date(2026, 8, 7, 16, 34, 0, 0, time.UTC),
			wantDone:      day(2026, 8, 7),
			wantNext:      day(2026, 10, 31),
		},
		{
			name: "every 4 weeks, starting 2 days before the deadline, with a weekday left over",
			rule: func() thingscloud.RepeaterConfiguration {
				r := afterCompletion(thingscloud.FrequencyUnitWeekly, 4)
				r.TimeShift = -2
				r.DetailConfiguration = []thingscloud.RepeaterDetailConfiguration{{Weekday: &sunday}}
				return r
			}(),
			withDeadlines: true,
			done:          time.Date(2026, 10, 30, 20, 1, 0, 0, time.UTC),
			wantDone:      day(2026, 10, 30),
			wantNext:      day(2026, 11, 25),
		},
		{
			name:     "every month, from the 31st",
			rule:     afterCompletion(thingscloud.FrequencyUnitMonthly, 1),
			done:     time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC),
			wantDone: day(2026, 1, 31),
			wantNext: day(2026, 2, 28),
		},
		{
			name:     "every year, from a leap day",
			rule:     afterCompletion(thingscloud.FrequencyUnitYearly, 1),
			done:     time.Date(2028, 2, 29, 12, 0, 0, 0, time.UTC),
			wantDone: day(2028, 2, 29),
			wantNext: day(2029, 2, 28),
		},
		{
			name: "every 2 weeks, ending after the next one",
			rule: func() thingscloud.RepeaterConfiguration {
				r := afterCompletion(thingscloud.FrequencyUnitWeekly, 2)
				end := thingscloud.Timestamp(day(2026, 12, 9))
				r.LastScheduledAt = &end
				return r
			}(),
			done:     time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC),
			wantDone: day(2026, 11, 25),
			wantNext: day(2026, 12, 9),
		},
	}
	for _, tt := range tests {
		for _, status := range []string{"completed", "canceled"} {
			t.Run(tt.name+"/"+status, func(t *testing.T) {
				inNewYorkAt(t, tt.done.Year(), tt.done.Month(), tt.done.Day(), tt.done.Hour(), tt.done.Minute())
				template := []taskOption{withTitle("Water plants"), withRule(tt.rule)}
				if tt.withDeadlines {
					template = append(template, withDeadline(day(4001, 1, 1)))
				}
				fc := newFakeCloud("test@example.com",
					makeTaskItem("template", template...),
					makeTaskItem("instance", withTitle("Water plants"), withTemplate("template")),
				)
				defer fc.Close()
				tmcp := newTestThingsMCP(t, fc)

				r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "instance", "status": status}))
				if err != nil {
					t.Fatal(err)
				}
				assertNotError(t, r)
				if commits := fc.getCommitLog(); len(commits) != 1 {
					t.Fatalf("posted %d commits, want both writes in one", len(commits))
				}
				written := lastCommit(t, fc)
				want := map[string]any{"acrd": float64(tt.wantDone.Unix()), "tir": float64(tt.wantNext.Unix())}
				if got := written["template"]; !reflect.DeepEqual(got, want) {
					t.Fatalf("template got %v, want %v (acrd %s, tir %s)", got, want, tt.wantDone.Format("2006-01-02"), tt.wantNext.Format("2006-01-02"))
				}
				wantStatus := float64(3)
				if status == "canceled" {
					wantStatus = 2
				}
				if got := written["instance"]; got["ss"] != wantStatus || got["sp"] == nil {
					t.Fatalf("instance got %v, want ss %v and a stop date", got, wantStatus)
				}
				next := tmcp.getState().Tasks["template"].TodayIndexRefDate
				if next == nil || !next.Equal(tt.wantNext) {
					t.Fatalf("local state's next date is %v, want %s", next, tt.wantNext.Format("2006-01-02"))
				}
			})
		}
	}
}

// Reopening a to-do that repeats after completion takes back its next one,
// and a to-do that's already done keeps the next date it set.
func TestReopeningARepeatAfterCompletion(t *testing.T) {
	tests := []struct {
		name         string
		from         thingscloud.TaskStatus
		to           string
		wantTemplate map[string]any
	}{
		{name: "reopen a completed one", from: thingscloud.TaskStatusCompleted, to: "pending", wantTemplate: map[string]any{"acrd": nil, "tir": nil}},
		{name: "reopen a canceled one", from: thingscloud.TaskStatusCanceled, to: "pending", wantTemplate: map[string]any{"acrd": nil, "tir": nil}},
		{name: "cancel a completed one", from: thingscloud.TaskStatusCompleted, to: "canceled"},
		{name: "complete a completed one", from: thingscloud.TaskStatusCompleted, to: "completed"},
		{name: "reopen an open one", from: thingscloud.TaskStatusPending, to: "pending"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := newFakeCloud("test@example.com",
				makeTaskItem("template", withTitle("Water plants"), withRule(afterCompletion(thingscloud.FrequencyUnitDaily, 3))),
				makeTaskItem("instance", withTitle("Water plants"), withTemplate("template"), withStatus(tt.from)),
			)
			defer fc.Close()
			tmcp := newTestThingsMCP(t, fc)

			r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "instance", "status": tt.to}))
			if err != nil {
				t.Fatal(err)
			}
			assertNotError(t, r)
			written := lastCommit(t, fc)
			got, wroteTemplate := written["template"]
			if tt.wantTemplate == nil {
				if wroteTemplate {
					t.Fatalf("wrote the template %v, want it left alone", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.wantTemplate) {
				t.Fatalf("template got %v, want %v", got, tt.wantTemplate)
			}
		})
	}
}

func TestRepeatsAfterCompletionThisCantSchedule(t *testing.T) {
	count := int64(5)
	soon := thingscloud.Timestamp(day(2026, 11, 30))
	tests := []struct {
		name     string
		rule     func(r *thingscloud.RepeaterConfiguration)
		template []taskOption
		instance []taskOption
		status   string
		want     string
	}{
		{name: "a set count", rule: func(r *thingscloud.RepeaterConfiguration) { r.RepeatCount = &count }, want: "a set number of times or until a date"},
		{name: "ends before the next one", rule: func(r *thingscloud.RepeaterConfiguration) { r.LastScheduledAt = &soon }, want: "a set number of times or until a date"},
		{name: "starts early without deadlines", rule: func(r *thingscloud.RepeaterConfiguration) { r.TimeShift = -2 }, want: "can't be read here"},
		{name: "starts after its deadline", rule: func(r *thingscloud.RepeaterConfiguration) { r.TimeShift = 2 }, template: []taskOption{withDeadline(day(4001, 1, 1))}, want: "can't be read here"},
		{name: "an unknown unit", rule: func(r *thingscloud.RepeaterConfiguration) { r.FrequencyUnit = 512 }, want: "can't be read here"},
		{name: "every 0 weeks", rule: func(r *thingscloud.RepeaterConfiguration) { r.FrequencyAmplitude = 0 }, want: "can't be read here"},
		{name: "an unknown rule version", rule: func(r *thingscloud.RepeaterConfiguration) { r.Version = 5 }, want: "can't be read here"},
		{name: "an unknown rule type", rule: func(r *thingscloud.RepeaterConfiguration) { r.Type = 2 }, want: "can't be read here"},
		{name: "repeat in the Trash", template: []taskOption{withTrashed()}, want: "can't be read here"},
		{name: "repeat finished", template: []taskOption{withStatus(thingscloud.TaskStatusCompleted)}, want: "can't be read here"},
		{name: "a repeating project", instance: []taskOption{withTaskType(thingscloud.TaskTypeProject)}, want: "this project repeats after completion"},
		{name: "trash", status: "trashed", want: "trashing or restoring it"},
		{name: "restore", instance: []taskOption{withTrashed(), withStatus(thingscloud.TaskStatusCompleted)}, status: "restored", want: "trashing or restoring it"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inNewYorkAt(t, 2026, 11, 25, 17, 0)
			rule := afterCompletion(thingscloud.FrequencyUnitWeekly, 1)
			if tt.rule != nil {
				tt.rule(&rule)
			}
			fc := newFakeCloud("test@example.com",
				makeTaskItem("template", append([]taskOption{withTitle("Water plants"), withRule(rule)}, tt.template...)...),
				makeTaskItem("instance", append([]taskOption{withTitle("Water plants"), withTemplate("template")}, tt.instance...)...),
			)
			defer fc.Close()
			tmcp := newTestThingsMCP(t, fc)

			status := tt.status
			if status == "" {
				status = "completed"
			}
			assertRefused(t, fc, func() (string, bool) {
				r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "instance", "status": status}))
				if err != nil {
					t.Fatal(err)
				}
				return resultText(t, r), r.IsError
			}, tt.want)
		})
	}
}

func TestOtherEditsOfARepeatingToDoAreWritten(t *testing.T) {
	fc := newFakeCloud("test@example.com",
		makeTaskItem("template", withTitle("Water plants"), withRepeat(1)),
		makeTaskItem("instance", withTitle("Water plants"), withTemplate("template")),
	)
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "instance", "title": "Water the ferns"}))
	if err != nil {
		t.Fatal(err)
	}
	assertNotError(t, r)
	written := lastCommit(t, fc)["instance"]
	if written["tt"] != "Water the ferns" || len(written) != 2 {
		t.Fatalf("wrote %v, want the title and modification date", written)
	}
}

func TestToDoWithANewerRepeatRuleIsNotEdited(t *testing.T) {
	fc := newFakeCloud("test@example.com", withNewerRepeatRule(t, makeTaskItem("task-1", withTitle("Water plants"))))
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)

	assertRefused(t, fc, func() (string, bool) {
		r, err := tmcp.handleEditTask(context.Background(), makeReq(map[string]any{"uuid": "task-1", "title": "Water the ferns"}))
		if err != nil {
			t.Fatal(err)
		}
		return resultText(t, r), r.IsError
	}, "format this server can't read")
}
