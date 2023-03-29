package allure

const StageFinished = "finished"

const (
	StatusPass   = "passed"
	StatusFail   = "failed"
	StatusSkip   = "skipped"
	StatusBroken = "broken"
)

type Test struct {
	UUID        string       `json:"uuid"`
	TestCaseID  string       `json:"testCaseId"`
	HistoryID   string       `json:"historyId"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	Stage       string       `json:"stage"`
	Steps       []Step       `json:"steps"`
	Start       int64        `json:"start"`
	Stop        int64        `json:"stop"`
	FullName    string       `json:"fullName"`
	Parameters  []Parameter  `json:"parameters"`
	Labels      []Label      `json:"labels"`
	Attachments []Attachment `json:"attachments"`
}

type Step struct {
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	Stage       string       `json:"stage"`
	Steps       []Step       `json:"steps"`
	Attachments []Attachment `json:"attachments"`
	Parameters  []Parameter  `json:"parameters"`
	Start       int64        `json:"start"`
	Stop        int64        `json:"stop"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Attachment struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Type   string `json:"type"`
}
