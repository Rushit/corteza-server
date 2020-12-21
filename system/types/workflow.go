package types

import (
	"github.com/cortezaproject/corteza-server/pkg/workflow"
	"time"
)

type (
	// Workflow represents entire workflow definition
	Workflow struct {
		ID      uint64            `json:"workflowID,string"`
		Name    string            `json:"name"`
		Handle  string            `json:"handle"`
		Labels  map[string]string `json:"labels,omitempty"`
		Meta    WorkflowMeta      `json:"meta"`
		Enabled bool              `json:"enabled"`

		Trace        bool          `json:"trace"`
		KeepSessions time.Duration `json:"keepSessions"`

		// Initial input scope
		Scope workflow.Variables `json:"scope"`

		OwnedBy   uint64     `json:"ownedBy,string"`
		CreatedAt time.Time  `json:"createdAt,omitempty"`
		CreatedBy uint64     `json:"createdBy,string" `
		UpdatedAt *time.Time `json:"updatedAt,omitempty"`
		UpdatedBy uint64     `json:"updatedBy,string,omitempty"`
		DeletedAt *time.Time `json:"deletedAt,omitempty"`
		DeletedBy uint64     `json:"deletedBy,string,omitempty"`
	}

	WorkflowMeta struct {
		Description string                 `json:"description"`
		Visual      map[string]interface{} `json:"visual"`
	}

	WorkflowTrigger struct {
		ID         uint64 `json:"triggerID,string"`
		WorkflowID uint64 `json:"workflowID,string"`
		Enabled    bool   `json:"enabled"`

		// Resource type that can trigger the workflow
		ResourceType string

		// Event type that can trigger the workflow
		EventType string

		// Trigger constraints
		Constraints interface{} // @todo

		Meta WorkflowTriggerMeta `json:"meta"`

		// Initial input scope,
		// will be merged merged with workflow variables
		Input workflow.Variables

		OwnedBy   uint64     `json:"ownedBy,string"`
		CreatedAt time.Time  `json:"createdAt,omitempty"`
		CreatedBy uint64     `json:"createdBy,string" `
		UpdatedAt *time.Time `json:"updatedAt,omitempty"`
		UpdatedBy uint64     `json:"updatedBy,string,omitempty"`
		DeletedAt *time.Time `json:"deletedAt,omitempty"`
		DeletedBy uint64     `json:"deletedBy,string,omitempty"`
	}

	WorkflowTriggerMeta struct {
		Description string                 `json:"description"`
		Visual      map[string]interface{} `json:"visual"`
	}

	//WorkflowMetrics struct {
	//	WorkflowID  uint64        `json:"workflowID,string"`
	//	SessionID   uint64        `json:"parentID,string"`
	//	Triggered   string        `json:"triggered,string"`   // event name
	//	TriggeredBy string        `json:"triggeredBy,string"` // resource ID
	//	ExecutedBy  uint64        `json:"executedBy,string"`
	//	ExecutedAt  time.Time     `json:"executedAt"`
	//	Duration    time.Duration `json:"duration"`
	//}

	// WorkflowStep describes one workflow step
	WorkflowStep struct {
		ID         uint64           `json:"stepID,string"`
		WorkflowID uint64           `json:"workflowID,string"`
		Meta       WorkflowStepMeta `json:"meta"`
		FunctionID uint64           `json:"functionID"`

		OwnedBy   uint64     `json:"ownedBy,string"`
		CreatedAt time.Time  `json:"createdAt,omitempty"`
		CreatedBy uint64     `json:"createdBy,string" `
		UpdatedAt *time.Time `json:"updatedAt,omitempty"`
		UpdatedBy uint64     `json:"updatedBy,string,omitempty"`
		DeletedAt *time.Time `json:"deletedAt,omitempty"`
		DeletedBy uint64     `json:"deletedBy,string,omitempty"`
	}

	WorkflowStepMeta struct {
		Description string                 `json:"description"`
		Visual      map[string]interface{} `json:"visual"`
	}

	// WorkflowPath defines connection between two workflow steps
	WorkflowPath struct {
		WorkflowID uint64 `json:"workflowID,string"`
		ParentID   uint64 `json:"parentID,string"`
		ChildID    uint64 `json:"childID,string"`

		// test expression for gateway paths
		Test *WorkflowExpression `json:"test,string"`
		Meta WorkflowPathMeta    `json:"meta"`
	}

	WorkflowPathMeta struct {
		Label       string                 `json:"description"`
		Description string                 `json:"description"`
		Visual      map[string]interface{} `json:"visual"`
	}

	// Instance of single workflow execution
	WorkflowSession struct {
		ID         uint64 `json:"sessionID,string"`
		WorkflowID uint64 `json:"workflowID,string"`

		Triggered   string        `json:"triggered,string"`   // event name
		TriggeredBy string        `json:"triggeredBy,string"` // resource ID
		ExecutedBy  uint64        `json:"executedBy,string"`  // runner (might be different then creator)
		WallTime    time.Duration `json:"wallTime"`           // how long did it take to run it (inc all suspension)
		UserTime    time.Duration `json:"userTime"`           // how long did it take to run it (sum of all time spent in each step)

		Input  workflow.Variables `json:"input"`
		Output workflow.Variables `json:"output"`

		Trace []WorkflowSessionTraceStep `json:"trace"`

		CreatedAt time.Time  `json:"createdAt,omitempty"`
		CreatedBy uint64     `json:"createdBy,string"`
		DeletedAt *time.Time `json:"deletedAt,omitempty"`
		DeletedBy uint64     `json:"deletedBy,string,omitempty"`
		PurgeAt   *time.Time `json:"purgeAt,omitempty"`
	}

	// WorkflowSessionTraceStep stores info and insturmentatino on visited workflow steps
	WorkflowSessionTraceStep struct {
		ID         uint64             `json:"traceStepID,string"`
		CallerStep uint64             `json:"traceCallerStepID,string"`
		WorkflowID uint64             `json:"workflowID,string"`
		StateID    uint64             `json:"stateID,string"`
		SessionID  uint64             `json:"sessionID,string"`
		CallerID   uint64             `json:"callerID,string"`
		StepID     uint64             `json:"stepID,string"`
		Depth      uint64             `json:"depth,string"`
		Scope      workflow.Variables `json:"scope"`
		Duration   time.Duration      `json:"duration"`
	}

	// WorkflowState tracks suspended sessions
	// Session can have more than one state
	WorkflowState struct {
		ID        uint64 `json:"stateID,string"`
		SessionID uint64 `json:"sessionID,string"`

		ResumeAt        *time.Time `json:"resumeAt"`
		WaitingForInput bool       `json:"waitingForInput"`

		CreatedAt time.Time `json:"createdAt,omitempty"`
		CreatedBy uint64    `json:"createdBy,string"`

		CallerID uint64             `json:"callerID,string"`
		StepID   uint64             `json:"stepID,string"`
		Scope    workflow.Variables `json:"scope"`
	}

	// @todo we need a better name
	//       internally this is a activity handler
	WorkflowFunction struct {
		ID        uint64                   `json:"functionID,string"`
		Name      string                   `json:"name"`
		Meta      WorkflowFunctionMeta     `json:"meta"`
		Handler   workflow.ActivityHandler `json:"-"`
		Arguments []*WorkflowExpression    `json:"arguments"`
		Results   []*WorkflowExpression    `json:"results"`
	}

	WorkflowFunctionMeta struct {
		Description string                 `json:"description"`
		Visual      map[string]interface{} `json:"visual"`
	}

	// Used for expression steps, arguments and results mapping
	WorkflowExpression struct {
		Name string `json:"name"`
		Expr string `json:"expr"`
	}
)
