package safety

import (
	"fmt"
	"strings"

	"github.com/dynatrace-oss/dtctl/pkg/config"
)

// Operation represents a type of operation that can be performed
type Operation string

const (
	// OperationRead is a read-only operation
	OperationRead Operation = "read"
	// OperationCreate is a create operation
	OperationCreate Operation = "create"
	// OperationUpdate is an update operation
	OperationUpdate Operation = "update"
	// OperationDelete is a delete operation
	OperationDelete Operation = "delete"
	// OperationDeleteBucket is a bucket deletion operation (data loss)
	OperationDeleteBucket Operation = "delete-bucket"
)

// ResourceOwnership indicates whether a resource is owned by the current user
type ResourceOwnership int

const (
	// OwnershipUnknown means ownership cannot be determined
	OwnershipUnknown ResourceOwnership = iota
	// OwnershipOwn means the resource is owned by the current user
	OwnershipOwn
	// OwnershipShared means the resource is owned by someone else
	OwnershipShared
)

// CheckResult contains the result of a safety check
type CheckResult struct {
	Allowed     bool
	Reason      string
	Suggestions []string
}

// Checker performs safety level checks for operations
type Checker struct {
	contextName string
	safetyLevel config.SafetyLevel
	overridden  bool
}

// NewChecker creates a new safety checker for a context
func NewChecker(contextName string, ctx *config.Context) *Checker {
	return &Checker{
		contextName: contextName,
		safetyLevel: ctx.GetEffectiveSafetyLevel(),
		overridden:  false,
	}
}

// NewCheckerWithLevel creates a new safety checker with an explicit safety level
func NewCheckerWithLevel(contextName string, level config.SafetyLevel) *Checker {
	return &Checker{
		contextName: contextName,
		safetyLevel: level,
		overridden:  false,
	}
}

// SetOverride marks that the safety check was overridden
func (c *Checker) SetOverride(overridden bool) {
	c.overridden = overridden
}

// IsOverridden returns whether the safety check was overridden
func (c *Checker) IsOverridden() bool {
	return c.overridden
}

// SafetyLevel returns the current safety level
func (c *Checker) SafetyLevel() config.SafetyLevel {
	return c.safetyLevel
}

// ContextName returns the context name
func (c *Checker) ContextName() string {
	return c.contextName
}

// Check verifies if an operation is allowed under the current safety level
func (c *Checker) Check(op Operation, ownership ResourceOwnership) CheckResult {
	// If overridden, always allow
	if c.overridden {
		return CheckResult{Allowed: true}
	}

	switch c.safetyLevel {
	case config.SafetyLevelReadOnly:
		return c.checkReadOnly(op)
	case config.SafetyLevelReadWriteMine:
		return c.checkReadWriteMine(op, ownership)
	case config.SafetyLevelReadWriteAll:
		return c.checkReadWriteAll(op)
	case config.SafetyLevelDangerouslyUnrestricted:
		return CheckResult{Allowed: true}
	default:
		// Unknown level, default to readwrite-mine behavior
		return c.checkReadWriteMine(op, ownership)
	}
}

func (c *Checker) checkReadOnly(op Operation) CheckResult {
	if op == OperationRead {
		return CheckResult{Allowed: true}
	}

	return CheckResult{
		Allowed: false,
		Reason:  fmt.Sprintf("Context '%s' (%s) does not allow %s operations", c.contextName, c.safetyLevel, op),
		Suggestions: []string{
			"Switch to a context with write permissions",
			"Use --override-safety to bypass this check",
		},
	}
}

func (c *Checker) checkReadWriteMine(op Operation, ownership ResourceOwnership) CheckResult {
	switch op {
	case OperationRead, OperationCreate:
		return CheckResult{Allowed: true}
	case OperationUpdate, OperationDelete:
		if ownership == OwnershipShared {
			return CheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Context '%s' (%s) does not allow modifying resources owned by others", c.contextName, c.safetyLevel),
				Suggestions: []string{
					"Switch to a 'readwrite-all' context",
					"Use --override-safety to bypass this check",
					"Use --assume-mine if you own this resource",
				},
			}
		}
		return CheckResult{Allowed: true}
	case OperationDeleteBucket:
		return CheckResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Context '%s' (%s) does not allow bucket deletion", c.contextName, c.safetyLevel),
			Suggestions: []string{
				"Bucket operations require 'dangerously-unrestricted' safety level",
				"Use --override-safety to bypass this check",
			},
		}
	}
	return CheckResult{Allowed: true}
}

func (c *Checker) checkReadWriteAll(op Operation) CheckResult {
	if op == OperationDeleteBucket {
		return CheckResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Context '%s' (%s) does not allow bucket deletion", c.contextName, c.safetyLevel),
			Suggestions: []string{
				"Bucket operations require 'dangerously-unrestricted' safety level",
				"Use --override-safety to bypass this check",
			},
		}
	}
	return CheckResult{Allowed: true}
}

// FormatError formats a CheckResult as an error message
func (c *Checker) FormatError(result CheckResult) string {
	if result.Allowed {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Operation not allowed:\n   Context: %s (%s)\n   Reason: %s", c.contextName, c.safetyLevel, result.Reason)

	if len(result.Suggestions) > 0 {
		b.WriteString("\n\nSuggestions:")
		for _, s := range result.Suggestions {
			fmt.Fprintf(&b, "\n  • %s", s)
		}
	}

	return b.String()
}

// OverrideWarning returns a warning message when safety is overridden
func (c *Checker) OverrideWarning(op Operation) string {
	return fmt.Sprintf("Safety check bypassed: %s operation normally requires higher safety level than '%s'", op, c.safetyLevel)
}

// CheckError performs a safety check and returns an error if not allowed
func (c *Checker) CheckError(op Operation, ownership ResourceOwnership) error {
	result := c.Check(op, ownership)
	if !result.Allowed {
		return fmt.Errorf("%s", c.FormatError(result))
	}
	return nil
}

// SafetyError represents a safety check failure
type SafetyError struct {
	ContextName string
	SafetyLevel config.SafetyLevel
	Operation   Operation
	Reason      string
	Suggestions []string
}

func (e *SafetyError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Operation not allowed:\n   Context: %s (%s)\n   Reason: %s",
		e.ContextName, e.SafetyLevel, e.Reason)

	if len(e.Suggestions) > 0 {
		b.WriteString("\n\nSuggestions:")
		for _, s := range e.Suggestions {
			fmt.Fprintf(&b, "\n  • %s", s)
		}
	}

	return b.String()
}
