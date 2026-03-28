package engine

import (
	"fmt"
	"iter"
)

// CheckRegistry holds all registered checks, keyed by name.
// It enforces uniqueness and provides iteration for config/wizard discovery.
type CheckRegistry struct {
	checks map[string]Check
	order  []string // insertion order for stable iteration
}

// NewCheckRegistry creates an empty CheckRegistry.
func NewCheckRegistry() *CheckRegistry {
	return &CheckRegistry{
		checks: make(map[string]Check),
	}
}

// Register adds a check to the registry. Panics if the name is already taken.
func (r *CheckRegistry) Register(c Check) {
	name := c.Name()
	if _, exists := r.checks[name]; exists {
		panic(fmt.Sprintf("engine: check %q already registered", name))
	}

	r.checks[name] = c
	r.order = append(r.order, name)
}

// Get returns a check by name.
func (r *CheckRegistry) Get(name string) (Check, bool) {
	c, ok := r.checks[name]

	return c, ok
}

// Len returns the number of registered checks.
func (r *CheckRegistry) Len() int {
	return len(r.checks)
}

// All iterates over all registered checks in insertion order.
func (r *CheckRegistry) All() iter.Seq2[string, Check] {
	return func(yield func(string, Check) bool) {
		for _, name := range r.order {
			if !yield(name, r.checks[name]) {
				return
			}
		}
	}
}

// ActionRegistry holds all registered actions, keyed by name.
type ActionRegistry struct {
	actions map[string]Action
	order   []string
}

// NewActionRegistry creates an empty ActionRegistry.
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		actions: make(map[string]Action),
	}
}

// Register adds an action to the registry. Panics if the name is already taken.
func (r *ActionRegistry) Register(a Action) {
	name := a.Name()
	if _, exists := r.actions[name]; exists {
		panic(fmt.Sprintf("engine: action %q already registered", name))
	}

	r.actions[name] = a
	r.order = append(r.order, name)
}

// Get returns an action by name.
func (r *ActionRegistry) Get(name string) (Action, bool) {
	a, ok := r.actions[name]

	return a, ok
}

// Len returns the number of registered actions.
func (r *ActionRegistry) Len() int {
	return len(r.actions)
}

// All iterates over all registered actions in insertion order.
func (r *ActionRegistry) All() iter.Seq2[string, Action] {
	return func(yield func(string, Action) bool) {
		for _, name := range r.order {
			if !yield(name, r.actions[name]) {
				return
			}
		}
	}
}
