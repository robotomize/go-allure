package goexec

import (
	"time"
)

const (
	GoTestActionOutput = "output"
	GoTestActionPass   = "pass"
	GoTestActionFail   = "fail"
	GoTestActionRun    = "run"
	GoTestActionCont   = "cont"
	GoTestActionPause  = "pause"
	GoTestActionSkip   = "skip"
	GoTestActionPanic  = "panic"
)

type GoTestEntry struct {
	Time     time.Time
	TestName string `json:"Test"`
	Action   string
	Package  string
	Elapsed  float64
	Output   string
}

type GoTest struct {
	Name    string
	Log     string
	Package string
	Stage   string
	Start   time.Time
	Stop    time.Time
	Status  string
	Elapsed time.Duration
	Output  []string
}
