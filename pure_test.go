package main

import (
	"hash/crc32"
	"testing"
	"time"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
)

// ---------------------------------------------------------------------------
// parseDate
// ---------------------------------------------------------------------------

func TestParseDate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantNil bool
		check  func(t *testing.T, got *time.Time)
	}{
		{
			name:  "YYYY-MM-DD returns UTC midnight",
			input: "2025-03-15",
			check: func(t *testing.T, got *time.Time) {
				want := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:  "RFC3339 with timezone",
			input: "2025-03-15T10:00:00+08:00",
			check: func(t *testing.T, got *time.Time) {
				want, _ := time.Parse(time.RFC3339, "2025-03-15T10:00:00+08:00")
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
		},
		{
			name:    "invalid string returns nil",
			input:   "bad",
			wantNil: true,
		},
		{
			name:    "partial date returns nil",
			input:   "2025-13-01",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDate(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseTime
// ---------------------------------------------------------------------------

func TestParseTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSecs  int
		wantValid bool
	}{
		{"09:30", "09:30", 34200, true},
		{"23:59", "23:59", 86340, true},
		{"00:00", "00:00", 0, true},
		{"invalid hour", "25:00", 0, false},
		{"invalid minute", "12:60", 0, false},
		{"no colon", "1200", 0, false},
		{"negative hour", "-1:00", 0, false},
		{"empty", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secs, valid := parseTime(tt.input)
			if valid != tt.wantValid {
				t.Errorf("valid: got %v, want %v", valid, tt.wantValid)
			}
			if valid && secs != tt.wantSecs {
				t.Errorf("secs: got %d, want %d", secs, tt.wantSecs)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// textNote / emptyNote — CRC32 correctness
// ---------------------------------------------------------------------------

func TestTextNote(t *testing.T) {
	text := "Hello, world!"
	note := textNote(text)

	if note.TypeTag != "tx" {
		t.Errorf("TypeTag: got %q, want %q", note.TypeTag, "tx")
	}
	if note.Type != 1 {
		t.Errorf("Type: got %d, want 1", note.Type)
	}
	if note.Value != text {
		t.Errorf("Value: got %q, want %q", note.Value, text)
	}
	expectedCRC := int64(crc32.ChecksumIEEE([]byte(text)))
	if note.Checksum != expectedCRC {
		t.Errorf("Checksum: got %d, want %d", note.Checksum, expectedCRC)
	}
}

func TestEmptyNote(t *testing.T) {
	note := emptyNote()

	if note.TypeTag != "tx" {
		t.Errorf("TypeTag: got %q, want %q", note.TypeTag, "tx")
	}
	if note.Checksum != 0 {
		t.Errorf("Checksum: got %d, want 0", note.Checksum)
	}
	if note.Value != "" {
		t.Errorf("Value: got %q, want empty", note.Value)
	}
	if note.Type != 1 {
		t.Errorf("Type: got %d, want 1", note.Type)
	}
}

// ---------------------------------------------------------------------------
// scheduleString
// ---------------------------------------------------------------------------

func TestScheduleString(t *testing.T) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name        string
		schedule    thingscloud.TaskSchedule
		date        *time.Time
		startBucket int
		want        string
	}{
		{"st=0 → inbox", thingscloud.TaskScheduleInbox, nil, 0, "inbox"},
		{"st=1 + no date → anytime", thingscloud.TaskScheduleAnytime, nil, 0, "anytime"},
		{"st=1 + past date → anytime", thingscloud.TaskScheduleAnytime, &past, 0, "anytime"},
		{"st=1 + future date → anytime", thingscloud.TaskScheduleAnytime, &future, 0, "anytime"},
		{"st=1 + today → today", thingscloud.TaskScheduleAnytime, &today, 0, "today"},
		{"st=1 + today + sb=1 → tonight", thingscloud.TaskScheduleAnytime, &today, 1, "tonight"},
		{"st=2 + no date → someday", thingscloud.TaskScheduleSomeday, nil, 0, "someday"},
		{"st=2 + date → upcoming", thingscloud.TaskScheduleSomeday, &future, 0, "upcoming"},
		{"unknown schedule → inbox", thingscloud.TaskSchedule(99), nil, 0, "inbox"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scheduleString(tt.schedule, tt.date, tt.startBucket)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// statusString
// ---------------------------------------------------------------------------

func TestStatusString(t *testing.T) {
	tests := []struct {
		status thingscloud.TaskStatus
		want   string
	}{
		{thingscloud.TaskStatusPending, "pending"},
		{thingscloud.TaskStatusCompleted, "completed"},
		{thingscloud.TaskStatusCanceled, "canceled"},
		{thingscloud.TaskStatus(99), "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := statusString(tt.status)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// noteChecksum
// ---------------------------------------------------------------------------

func TestNoteChecksum(t *testing.T) {
	text := "test note"
	got := noteChecksum(text)
	want := int64(crc32.ChecksumIEEE([]byte(text)))
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

// ---------------------------------------------------------------------------
// offsetToTime
// ---------------------------------------------------------------------------

func TestOffsetToTime(t *testing.T) {
	tests := []struct {
		secs int
		want string
	}{
		{34200, "09:30"},
		{0, "00:00"},
		{86340, "23:59"},
		{3600, "01:00"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := offsetToTime(tt.secs)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
