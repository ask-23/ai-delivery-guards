package report

type Severity string

const (
	Pass Severity = "PASS"
	Warn Severity = "WARN"
	Fail Severity = "FAIL"
)

type Finding struct {
	Severity Severity
	Check    string
	Message  string
}
