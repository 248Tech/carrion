package actions

import "time"

// Action is the interface for all policy-emitted actions.
type Action interface {
	ID() string
	Timestamp() time.Time
	InstanceName() string
	Reason() string
	Type() string
}

// Base holds common action fields.
type Base struct {
	ActionID      string
	ActionTime    time.Time
	Instance      string
	ReasonText    string
	ActionType    string
}

func (b Base) ID() string              { return b.ActionID }
func (b Base) Timestamp() time.Time    { return b.ActionTime }
func (b Base) InstanceName() string     { return b.Instance }
func (b Base) Reason() string          { return b.ReasonText }
func (b Base) Type() string            { return b.ActionType }

// SetGamePref sets a game preference.
type SetGamePref struct {
	Base
	Pref  string
	Value string
}

// Say sends a chat message.
type Say struct {
	Base
	Message string
}

// RestoreBaseline restores all prefs to baseline.
type RestoreBaseline struct {
	Base
}

// Noop is a no-op (e.g. no state change needed).
type Noop struct {
	Base
}

// NewSetGamePref creates a SetGamePref action.
func NewSetGamePref(id, instance, reason, pref, value string) *SetGamePref {
	return &SetGamePref{
		Base: Base{
			ActionID:   id,
			ActionTime: time.Now(),
			Instance:   instance,
			ReasonText: reason,
			ActionType: "SetGamePref",
		},
		Pref:  pref,
		Value: value,
	}
}

// NewSay creates a Say action.
func NewSay(id, instance, reason, message string) *Say {
	return &Say{
		Base: Base{
			ActionID:   id,
			ActionTime: time.Now(),
			Instance:   instance,
			ReasonText: reason,
			ActionType: "Say",
		},
		Message: message,
	}
}

// NewRestoreBaseline creates a RestoreBaseline action.
func NewRestoreBaseline(id, instance, reason string) *RestoreBaseline {
	return &RestoreBaseline{
		Base: Base{
			ActionID:   id,
			ActionTime: time.Now(),
			Instance:   instance,
			ReasonText: reason,
			ActionType: "RestoreBaseline",
		},
	}
}

// NewNoop creates a Noop action.
func NewNoop(id, instance, reason string) *Noop {
	return &Noop{
		Base: Base{
			ActionID:   id,
			ActionTime: time.Now(),
			Instance:   instance,
			ReasonText: reason,
			ActionType: "Noop",
		},
	}
}
