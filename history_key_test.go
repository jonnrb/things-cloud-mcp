package main

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
)

// A history key alone reads and writes the account's Things data, so neither
// the log nor a diagnosis, which is kept and shared by URL, may carry it.
func TestHistoryKeyStaysOutOfLogsAndDiagnoses(t *testing.T) {
	var logs bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(old) })

	fc := newFakeCloud("test@example.com", makeTaskItem("task-1", withTitle("Visible")))
	defer fc.Close()
	tmcp := newTestThingsMCP(t, fc)
	report := tmcp.handleDiagnose(fc.email, "testpass")
	if report.Summary.Failed != 0 {
		t.Fatalf("diagnosis failed: %v", report.Errors)
	}
	diagnosis, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	for name, text := range map[string]string{"log": logs.String(), "diagnosis": string(diagnosis)} {
		if strings.Contains(text, fc.historyID) {
			t.Errorf("%s carries the history key:\n%s", name, text)
		}
	}
	fp := thingscloud.HistoryFingerprint(fc.historyID)
	if !strings.Contains(string(diagnosis), fp) {
		t.Errorf("diagnosis doesn't name the history by fingerprint %s:\n%s", fp, diagnosis)
	}
	if !strings.Contains(logs.String(), "Authoritative history: "+fp) {
		t.Errorf("log doesn't name the history by fingerprint %s:\n%s", fp, logs.String())
	}
}
