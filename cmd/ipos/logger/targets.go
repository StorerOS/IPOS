package logger

type Target interface {
	Send(entry interface{}, errKind string) error
}

var Targets = []Target{}

func AddTarget(t Target) {
	Targets = append(Targets, t)
}
