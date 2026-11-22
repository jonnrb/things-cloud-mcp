package memory

import (
	"encoding/json"
	"testing"

	things "github.com/arthursoares/things-cloud-sdk"
)

func stringVal(str string) *string {
	return &str
}

var newAreaPayload = `{
  "tt": "New Area"
}`

var newTagPayload = `{
  "ix": 4,
  "tt": "‼",
  "sh": "",
  "pn": []
}`

var newCheckListItemPayload = `{
  "md": 1495650756.177454,
  "ix": -543,
  "ss": 0,
  "tt": "A CheckListItem",
  "sp": null,
  "ts": [
    "1452D14B-3099-4251-B7C8-8B4C2B9BA334"
  ],
  "cd": 1495650723.3711741
}`

var newTaskPayload = `{
  "ix": -485,
  "cd": 1495650251.230479,
  "icsd": null,
  "ar": [],
  "tir": null,
  "rt": [],
  "rr": null,
  "icc": 0,
  "tt": "A \"Task\"",
  "tr": false,
  "tp": 0,
  "acrd": null,
  "ti": 0,
  "tg": [],
  "icp": false,
  "nt": null,
  "do": 0,
  "dl": [],
  "lai": null,
  "dd": null,
  "pr": [
    "BB7323E0-36E5-4DE7-8203-6A27B67C1CD4"
  ],
  "md": 1495650259.683249,
  "ss": 0,
  "sr": null,
  "sp": null,
  "st": 1,
  "dds": null,
  "ato": null,
  "sb": 0,
  "agr": []
}`

func TestState_Update(t *testing.T) {
	t.Run("Empty State", func(t *testing.T) {
		t.Parallel()
		t.Run("Create Area", func(t *testing.T) {
			t.Parallel()
			s := NewState()
			if err := s.Update(things.Item{
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindArea,
				P:      json.RawMessage(newAreaPayload),
			}); err != nil {
				t.Fatal(err.Error())
			}

			if len(s.Areas) != 1 {
				t.Fatal("Expected to have a new area")
			}
		})

		t.Run("Create Tag", func(t *testing.T) {
			t.Parallel()
			s := NewState()
			if err := s.Update(things.Item{
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindTag,
				P:      json.RawMessage(newTagPayload),
			}); err != nil {
				t.Fatal(err.Error())
			}

			if len(s.Tags) != 1 {
				t.Fatal("Expected to have a new tag")
			}
		})

		t.Run("Create CheckListItem", func(t *testing.T) {
			t.Parallel()
			s := NewState()
			if err := s.Update(things.Item{
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindChecklistItem,
				P:      json.RawMessage(newCheckListItemPayload),
			}); err != nil {
				t.Fatal(err.Error())
			}

			if len(s.CheckListItems) != 1 {
				t.Fatal("Expected to have a new check list item")
			}
		})

		t.Run("Create Task", func(t *testing.T) {
			t.Parallel()
			s := NewState()
			if err := s.Update(things.Item{
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindTask,
				P:      json.RawMessage(newTaskPayload),
			}); err != nil {
				t.Fatal(err.Error())
			}

			if len(s.Tasks) != 1 {
				t.Fatal("Expected to have a new task")
			}
		})
	})

	t.Run("Existing State", func(t *testing.T) {
		t.Parallel()
		newState := func() *State {
			s := NewState()
			s.Update(things.Item{
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindArea,
				P:      json.RawMessage(newAreaPayload),
			}, things.Item{
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindTag,
				P:      json.RawMessage(newTagPayload),
			}, things.Item{
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindChecklistItem,
				P:      json.RawMessage(newCheckListItemPayload),
			}, things.Item{
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindTask,
				P:      json.RawMessage(newTaskPayload),
			})
			return s
		}

		t.Run("Update Task", func(t *testing.T) {
			t.Parallel()
			s := newState()
			if err := s.Update(things.Item{
				Action: things.ItemActionModified,
				Kind:   things.ItemKindTask,
				P:      json.RawMessage(`{"tt": "Modified Title"}`),
			}); err != nil {
				t.Fatal(err.Error())
			}
			if s.Tasks[""].Title != "Modified Title" {
				t.Fatal("Expected title to be updated")
			}
		})

		t.Run("Delete Task", func(t *testing.T) {
			t.Parallel()
			s := newState()
			if err := s.Update(things.Item{
				Action: things.ItemActionDeleted,
				Kind:   things.ItemKindTask,
				P:      json.RawMessage(`{}`),
			}); err != nil {
				t.Fatal(err.Error())
			}
			if len(s.Tasks) != 0 {
				t.Fatal("Expected to have no more tasks")
			}
		})

		t.Run("Update CheckListItem", func(t *testing.T) {
			t.Parallel()
			s := newState()
			if err := s.Update(things.Item{
				Action: things.ItemActionModified,
				Kind:   things.ItemKindChecklistItem,
				P:      json.RawMessage(`{"tt": "Modified Title"}`),
			}); err != nil {
				t.Fatal(err.Error())
			}
			if s.CheckListItems[""].Title != "Modified Title" {
				t.Fatal("Expected title to be updated")
			}
		})

		t.Run("Delete ChecklistItem", func(t *testing.T) {
			t.Parallel()
			s := newState()
			if err := s.Update(things.Item{
				Action: things.ItemActionDeleted,
				Kind:   things.ItemKindChecklistItem,
				P:      json.RawMessage(`{}`),
			}); err != nil {
				t.Fatal(err.Error())
			}
			if len(s.CheckListItems) != 0 {
				t.Fatal("Expected to have no more check list items")
			}
		})

		t.Run("Update Tag", func(t *testing.T) {
			t.Parallel()
			s := newState()
			if err := s.Update(things.Item{
				Action: things.ItemActionModified,
				Kind:   things.ItemKindTag,
				P:      json.RawMessage(`{"tt": "Modified Tag"}`),
			}); err != nil {
				t.Fatal(err.Error())
			}
			if s.Tags[""].Title != "Modified Tag" {
				t.Fatal("Expected title to be updated")
			}
		})

		t.Run("Delete Tag", func(t *testing.T) {
			t.Parallel()
			s := newState()
			if err := s.Update(things.Item{
				Action: things.ItemActionDeleted,
				Kind:   things.ItemKindTag,
				P:      json.RawMessage(`{}`),
			}); err != nil {
				t.Fatal(err.Error())
			}
			if len(s.Tags) != 0 {
				t.Fatal("Expected to have no more tags")
			}
		})

		t.Run("Update Area", func(t *testing.T) {
			t.Parallel()
			s := newState()

			if err := s.Update(things.Item{
				Action: things.ItemActionModified,
				Kind:   things.ItemKindArea,
				P:      json.RawMessage(`{"tt": "Modified Area"}`),
			}); err != nil {
				t.Fatal(err.Error())
			}

			if s.Areas[""].Title != "Modified Area" {
				t.Fatal("Expected title to be updated")
			}
		})

		t.Run("Delete Area", func(t *testing.T) {
			t.Parallel()
			s := newState()

			if err := s.Update(things.Item{
				Action: things.ItemActionDeleted,
				Kind:   things.ItemKindArea,
				P:      json.RawMessage(`{}`),
			}); err != nil {
				t.Fatal(err.Error())
			}

			if len(s.Areas) != 0 {
				t.Fatal("Expected to have no more areas")
			}
		})
	})
}

func TestState_Tombstone(t *testing.T) {
	t.Parallel()

	for _, kind := range []things.ItemKind{things.ItemKindTombstone} {
		t.Run(string(kind)+" deletes existing task", func(t *testing.T) {
			t.Parallel()
			s := NewState()

			taskUUID := things.EncodeLegacyIdentifier("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE")

			// Step 1: Create a task
			if err := s.Update(things.Item{
				UUID:   taskUUID,
				Action: things.ItemActionCreated,
				Kind:   things.ItemKindTask,
				P:      json.RawMessage(newTaskPayload),
			}); err != nil {
				t.Fatal(err.Error())
			}

			// Verify the task exists
			if _, ok := s.Tasks[taskUUID]; !ok {
				t.Fatal("Expected task to exist after creation")
			}

			// Step 2: Create a tombstone item referencing the task UUID.
			tombstonePayload := `{"dloid":"` + taskUUID + `","dld":1495662927.014228}`
			if err := s.Update(things.Item{
				UUID:   "TOMBSTONE-UUID",
				Action: things.ItemActionCreated,
				Kind:   kind,
				P:      json.RawMessage(tombstonePayload),
			}); err != nil {
				t.Fatal(err.Error())
			}

			// Step 3: Verify the task is deleted
			if _, ok := s.Tasks[taskUUID]; ok {
				t.Fatal("Expected task to be deleted after tombstone")
			}
		})
	}
}

func TestState_updateTag(t *testing.T) {
	s := NewState()
	a := &things.Tag{
		UUID:  "CC-Things-Tag-High",
		Title: "High",
	}
	b := &things.Tag{
		UUID:  "CC-Things-Tag-Priority",
		Title: "!!",
	}
	c := &things.Tag{
		UUID:  "CC-Things-Tag-Errand",
		Title: "Errand",
	}
	s.Tags[a.UUID] = a
	s.Tags[b.UUID] = b
	s.Tags[c.UUID] = c

	t.Run("sets Title", func(t *testing.T) {
		tag := s.updateTag(things.TagActionItem{
			Item: things.Item{UUID: a.UUID},
			P: things.TagActionItemPayload{
				Title: stringVal("a title"),
			},
		})
		if tag.Title != "a title" {
			t.Fatalf("Expected Title %q but got %q", "a title", tag.Title)
		}
	})

	t.Run("hierarchy", func(t *testing.T) {
		t.Run("create", func(t *testing.T) {
			a2 := s.updateTag(things.TagActionItem{
				Item: things.Item{UUID: a.UUID},
				P: things.TagActionItemPayload{
					ParentTagIDs: &[]string{b.UUID},
				},
			})
			if a2.ParentTagIDs[0] != b.UUID {
				t.Fatalf("Expected parent of %q, but got %q", b.UUID, a2.ParentTagIDs)
			}
		})
		t.Run("change", func(t *testing.T) {
			a2 := s.updateTag(things.TagActionItem{
				Item: things.Item{UUID: a.UUID},
				P: things.TagActionItemPayload{
					ParentTagIDs: &[]string{c.UUID},
				},
			})
			if a2.ParentTagIDs[0] != c.UUID {
				t.Fatalf("Expected parent of %q, but got %q", c.UUID, a2.ParentTagIDs)
			}
		})
		t.Run("no change", func(t *testing.T) {
			a2 := s.updateTag(things.TagActionItem{
				Item: things.Item{UUID: a.UUID},
				P:    things.TagActionItemPayload{},
			})
			if a2.ParentTagIDs[0] != c.UUID {
				t.Fatalf("Expected parent of %q, but got %q", c.UUID, a2.ParentTagIDs)
			}
		})
		t.Run("delete", func(t *testing.T) {
			a2 := s.updateTag(things.TagActionItem{
				Item: things.Item{UUID: a.UUID},
				P: things.TagActionItemPayload{
					ParentTagIDs: &[]string{},
				},
			})
			if len(a2.ParentTagIDs) != 0 {
				t.Fatalf("Expected parent of nil, but got %q", a2.ParentTagIDs)
			}
		})
	})
}

func TestState_Headings(t *testing.T) {
	t.Parallel()
	s := NewState()

	projType := things.TaskTypeProject
	headType := things.TaskTypeHeading
	taskType := things.TaskTypeTask

	// Create a project
	projPayload, _ := json.Marshal(things.TaskActionItemPayload{
		Title: stringVal("My Project"),
		Type:  &projType,
	})
	// Create a heading under the project
	headPayload, _ := json.Marshal(things.TaskActionItemPayload{
		Title:         stringVal("Phase 1"),
		Type:          &headType,
		ParentTaskIDs: &[]string{"proj-1"},
	})
	// Create a task under the heading
	taskPayload, _ := json.Marshal(things.TaskActionItemPayload{
		Title:          stringVal("Do something"),
		Type:           &taskType,
		ActionGroupIDs: &[]string{"head-1"},
		ParentTaskIDs:  &[]string{"proj-1"},
	})

	s.Update(
		things.Item{UUID: "proj-1", Kind: things.ItemKindTask, Action: things.ItemActionCreated, P: projPayload},
		things.Item{UUID: "head-1", Kind: things.ItemKindTask, Action: things.ItemActionCreated, P: headPayload},
		things.Item{UUID: "task-1", Kind: things.ItemKindTask, Action: things.ItemActionCreated, P: taskPayload},
	)

	headings := s.Headings("proj-1")
	if len(headings) != 1 {
		t.Fatalf("expected 1 heading, got %d", len(headings))
	}
	if headings[0].Title != "Phase 1" {
		t.Errorf("expected 'Phase 1', got '%s'", headings[0].Title)
	}

	tasks := s.TasksByHeading("head-1", ListOption{})
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task under heading, got %d", len(tasks))
	}
	if tasks[0].Title != "Do something" {
		t.Errorf("expected 'Do something', got '%s'", tasks[0].Title)
	}
}

func TestState_TasksWithoutArea_DeletedParent(t *testing.T) {
	t.Parallel()
	s := NewState()

	taskType := things.TaskTypeTask

	// Create a child task that references a parent that doesn't exist in state
	// (simulates parent deleted by tombstone or ItemActionDeleted)
	childPayload, _ := json.Marshal(things.TaskActionItemPayload{
		Title:         stringVal("orphaned child"),
		Type:          &taskType,
		ParentTaskIDs: &[]string{"deleted-parent-id"},
	})
	s.Update(
		things.Item{UUID: "child-1", Kind: things.ItemKindTask, Action: things.ItemActionCreated, P: childPayload},
	)

	// This should NOT panic even though "deleted-parent-id" is not in state.Tasks
	result := s.TasksWithoutArea()
	_ = result // just verify no panic
}

func TestStateUpdateRejectsMalformedKnownItem(t *testing.T) {
	s := NewState()
	err := s.Update(things.Item{
		UUID:   "broken-task",
		Kind:   things.ItemKindTask,
		Action: things.ItemActionCreated,
		P:      []byte(`{"tt":`),
	})
	if err == nil {
		t.Fatal("expected malformed payload error")
	}
	if len(s.Tasks) != 0 {
		t.Fatalf("state changed after malformed payload: %#v", s.Tasks)
	}
}

func TestStateUpdateRejectsUnknownKind(t *testing.T) {
	s := NewState()
	err := s.Update(things.Item{
		UUID:   "future-item",
		Kind:   things.ItemKind("Task8"),
		Action: things.ItemActionCreated,
		P:      []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected unknown kind error")
	}
}

func TestStateUpdateIgnoresVersionedSettings(t *testing.T) {
	s := NewState()
	for _, kind := range []things.ItemKind{things.ItemKindSettings, "Settings5", "Settings42"} {
		err := s.Update(things.Item{
			UUID:   "settings-" + string(kind),
			Kind:   kind,
			Action: things.ItemAction(99),
			P:      []byte(`not-json`),
		})
		if err != nil {
			t.Fatalf("settings kind %q blocked state update: %v", kind, err)
		}
	}
	if len(s.Tasks) != 0 || len(s.Areas) != 0 || len(s.Tags) != 0 || len(s.CheckListItems) != 0 {
		t.Fatalf("settings changed task graph: %#v", s)
	}
}

func TestStateUpdateIgnoresCommandRecords(t *testing.T) {
	s := NewState()
	err := s.Update(
		things.Item{
			UUID:   "99265a7f-72fe-4d9c-864a-5634022361b2",
			Kind:   things.ItemKindCommand,
			Action: things.ItemActionCreated,
			P:      []byte(`{"anything":"goes"}`),
		},
		things.Item{
			UUID:   "command-unknown-action",
			Kind:   things.ItemKindCommand,
			Action: things.ItemAction(99),
			P:      []byte(`not-json`),
		},
	)
	if err != nil {
		t.Fatalf("Command record blocked state update: %v", err)
	}
	if len(s.Tasks) != 0 || len(s.Areas) != 0 || len(s.Tags) != 0 || len(s.CheckListItems) != 0 {
		t.Fatalf("Command record changed task graph: %#v", s)
	}
}

func TestStateUpdateReadsTask7(t *testing.T) {
	const id = "KXJnBwmWiWyVqd8YKR8wvm"
	s := NewState()
	err := s.Update(
		things.Item{
			UUID:   id,
			Kind:   things.ItemKindTask7,
			Action: things.ItemActionCreated,
			P:      []byte(`{"tt":"Written by Things 3.23","ss":0,"st":1,"tp":0,"cd":1763100000,"md":null,"sr":null,"tir":null,"dd":null,"rr":null,"rt":[],"tr":false,"ar":[],"pr":[],"tg":[]}`),
		},
		// Older apps and this server still write Task6 modifications onto the
		// same task.
		things.Item{
			UUID:   id,
			Kind:   things.ItemKindTask,
			Action: things.ItemActionModified,
			P:      []byte(`{"tt":"Renamed by a Task6 writer","md":1763100100}`),
		},
	)
	if err != nil {
		t.Fatalf("Task7 blocked state update: %v", err)
	}
	task, ok := s.Tasks[id]
	if !ok {
		t.Fatalf("Task7 task missing from state: %#v", s.Tasks)
	}
	if task.Title != "Renamed by a Task6 writer" {
		t.Fatalf("title = %q, want the Task6 rename", task.Title)
	}
}

func TestStateUpdateRejectsMalformedTask7(t *testing.T) {
	s := NewState()
	err := s.Update(things.Item{
		UUID:   "KXJnBwmWiWyVqd8YKR8wvm",
		Kind:   things.ItemKindTask7,
		Action: things.ItemActionCreated,
		P:      []byte(`{"tt":`),
	})
	if err == nil {
		t.Fatal("expected malformed Task7 payload error")
	}
	if len(s.Tasks) != 0 {
		t.Fatalf("state changed after malformed Task7 payload: %#v", s.Tasks)
	}
}

func TestStateUpdateAppliesNativeNoteDeltas(t *testing.T) {
	const id = "KXJnBwmWiWyVqd8YKR8wvm"
	s := NewState()
	err := s.Update(
		things.Item{
			UUID:   id,
			Kind:   things.ItemKindTask7,
			Action: things.ItemActionCreated,
			P:      []byte(`{"tt":"Native task","nt":{"_t":"tx","t":1,"ch":4002128608,"v":"Native Task7 note α 🚀\nSecond line."}}`),
		},
		// What Things 3.23.3 wrote when "Second" was edited to "Updated".
		things.Item{
			UUID:   id,
			Kind:   things.ItemKindTask7,
			Action: things.ItemActionModified,
			P:      []byte(`{"nt":{"_t":"tx","t":2,"ps":[{"r":"Update","p":26,"l":5,"ch":3672733299}]}}`),
		},
	)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, want := s.Tasks[id].Note, "Native Task7 note α 🚀\nUpdated line."; got != want {
		t.Fatalf("note = %q, want %q", got, want)
	}
	if s.UnverifiedNoteEdits != 0 {
		t.Fatalf("UnverifiedNoteEdits = %d, want 0", s.UnverifiedNoteEdits)
	}

	err = s.Update(things.Item{
		UUID:   id,
		Kind:   things.ItemKindTask7,
		Action: things.ItemActionModified,
		P:      []byte(`{"nt":{"_t":"tx","t":2,"ps":[{"r":"!","p":0,"l":0,"ch":1}]}}`),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if s.UnverifiedNoteEdits != 1 {
		t.Fatalf("UnverifiedNoteEdits = %d after a patch with an unmatched checksum, want 1", s.UnverifiedNoteEdits)
	}
}

func TestStateUpdateTracksTheNewerRepeatRule(t *testing.T) {
	const id = "KXJnBwmWiWyVqd8YKR8wvm"
	s := NewState()
	update := func(action things.ItemAction, p string) {
		t.Helper()
		if err := s.Update(things.Item{UUID: id, Kind: things.ItemKindTask7, Action: action, P: []byte(p)}); err != nil {
			t.Fatalf("Update(%s): %v", p, err)
		}
	}

	update(things.ItemActionCreated, `{"tt":"Plain","rr":null,"rp":null,"rt":[]}`)
	if s.Tasks[id].ModernRepeater {
		t.Fatal("a null rp counted as a repeat rule")
	}
	update(things.ItemActionModified, `{"rp":"opaque"}`)
	if !s.Tasks[id].ModernRepeater {
		t.Fatal("a non-null rp didn't count as a repeat rule")
	}
	update(things.ItemActionModified, `{"tt":"Renamed"}`)
	if !s.Tasks[id].ModernRepeater {
		t.Fatal("an edit without rp cleared the repeat rule")
	}
	update(things.ItemActionModified, `{"rp":null}`)
	if s.Tasks[id].ModernRepeater {
		t.Fatal("rp set to null kept the repeat rule")
	}
}
