package workflow

import "context"

type (
	Steps []Step
	Step  interface {
		Exec(context.Context, *ExecRequest) (ExecResponse, error)
	}

	// list of workflow steps with relations
	workflow struct {
		steps    []Step
		children map[Step][]Step
		parents  map[Step][]Step
	}
)

func Workflow() *workflow {
	wf := &workflow{
		steps:    make([]Step, 0, 1024),
		children: make(map[Step][]Step),
		parents:  make(map[Step][]Step),
	}

	return wf
}

func (wf *workflow) AddStep(s Step, cc ...Step) {
	wf.steps = append(wf.steps, s)
	wf.children[s] = cc
	for _, c := range cc {
		wf.AddParent(c, s)
	}
}

func (wf *workflow) AddParent(c, p Step) {
	wf.parents[c] = append(wf.parents[c], p)
}

func (wf *workflow) Children(s Step) Steps {
	return wf.children[s]
}

func (wf *workflow) Parents(s Step) Steps {
	return wf.parents[s]
}
