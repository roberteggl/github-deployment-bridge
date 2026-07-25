// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package deployment

// Phase is a GitHub Deployment status state.
type Phase string

const (
	PhaseQueued     Phase = "queued"
	PhaseInProgress Phase = "in_progress"
	PhaseSuccess    Phase = "success"
	PhaseFailure    Phase = "failure"
	PhaseError      Phase = "error"
	PhaseInactive   Phase = "inactive"
)

// StatusDescription returns the GitHub status description for a phase.
func StatusDescription(p Phase) string {
	switch p {
	case PhaseQueued:
		return "Deployment queued by FluxCD."
	case PhaseInProgress:
		return "Deployment in progress."
	case PhaseSuccess:
		return "Deployment completed successfully."
	case PhaseFailure:
		return "Deployment failed."
	case PhaseError:
		return "Deployment reporting failed."
	case PhaseInactive:
		return "Superseded by newer deployment."
	default:
		return ""
	}
}

// CanTransition reports whether moving from → to is a legal lifecycle step.
// Identical states are not transitions (callers treat those as duplicates).
func CanTransition(from, to Phase) bool {
	if from == to {
		return false
	}
	if from == "" {
		return true
	}
	switch from {
	case PhaseQueued:
		return to == PhaseInProgress || to == PhaseSuccess || to == PhaseFailure || to == PhaseError
	case PhaseInProgress:
		return to == PhaseSuccess || to == PhaseFailure || to == PhaseError
	case PhaseSuccess:
		return to == PhaseInactive
	case PhaseFailure, PhaseError, PhaseInactive:
		return false
	default:
		return false
	}
}

// IsTerminal reports whether a phase is terminal for its deployment key
// (no further application statuses except inactive supersession for success).
func IsTerminal(p Phase) bool {
	return p == PhaseSuccess || p == PhaseFailure || p == PhaseError || p == PhaseInactive
}

// benignSkippedTransition reports whether refusing from → to is expected
// (e.g. Flux re-reconcile after success briefly looking in_progress).
func benignSkippedTransition(from, to Phase) bool {
	return IsTerminal(from) && (to == PhaseQueued || to == PhaseInProgress)
}
