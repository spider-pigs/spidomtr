package testunit

// TestOutcome type
type TestOutcome int

// Fail is a failed test
const Fail TestOutcome = 0

// Skip is a skipped test
const Skip TestOutcome = 1

// Pass is a passed test
const Pass TestOutcome = 2

func (x TestOutcome) String() string {
	switch x {
	case Fail:
		return "fail"
	case Skip:
		return "skip"
	case Pass:
		return "pass"
	}
	panic("unkown type")
}
