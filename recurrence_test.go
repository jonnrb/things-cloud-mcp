package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

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
		refused  string
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
			refused: "repeats after completion",
		},
		{
			name: "template missing",
			template: func(*testing.T) []thingscloud.Item {
				return nil
			},
			refused: "repeats after completion",
		},
		{
			name: "template has a newer repeat rule",
			template: func(t *testing.T) []thingscloud.Item {
				return []thingscloud.Item{withNewerRepeatRule(t, makeTaskItem("template", withTitle("Water plants"), withRepeat(0)))}
			},
			refused: "repeats after completion",
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
				if tt.refused != "" {
					assertRefused(t, fc, call, tt.refused)
					return
				}
				if text, isError := call(); isError {
					t.Fatalf("status change refused: %s", text)
				}
				written := lastCommit(t, fc)
				if _, ok := written["template"]; ok || len(written) != 1 {
					t.Fatalf("wrote %v, want only the instance", written)
				}
			})
		}
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
	}, "newer format")
}
