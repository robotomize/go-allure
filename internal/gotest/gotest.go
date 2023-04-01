package gotest

import (
	"time"
)

const (
	ActionOutput = "output"
	ActionPass   = "pass"
	ActionFail   = "fail"
	ActionRun    = "run"
	ActionCont   = "cont"
	ActionPause  = "pause"
	ActionSkip   = "skip"
	ActionPanic  = "panic"
)

type Entry struct {
	Time     time.Time
	TestName string `json:"Test"`
	Action   string
	Package  string
	Elapsed  float64
	Output   string
}

type Test struct {
	Name    string
	Package string
	Stage   string
	Start   time.Time
	Stop    time.Time
	Status  string
	Elapsed time.Duration
	Output  []string
}

func (t *Test) FullName() string {
	return t.Package + "/" + t.Name
}

func (t *Test) Update(row Entry) {
	switch row.Action {
	case ActionCont:
		t.Stage = ActionCont
	case ActionSkip:
		t.Stop = row.Time
		t.Status = ActionSkip
		t.Stage = ActionSkip
		t.Elapsed = t.Stop.Sub(t.Start)
	case ActionFail:
		t.Stop = row.Time
		t.Status = ActionFail
		t.Stage = ActionFail
		t.Elapsed = t.Stop.Sub(t.Start)
	case ActionOutput:
		t.Output = append(t.Output, row.Output)
	case ActionPass:
		t.Stop = row.Time
		t.Status = ActionPass
		t.Stage = ActionPass
		t.Elapsed = t.Stop.Sub(t.Start)
	case ActionPause:
		t.Stage = ActionPause
	case ActionRun:
		t.Start = row.Time
		t.Stage = ActionRun
	}
}
