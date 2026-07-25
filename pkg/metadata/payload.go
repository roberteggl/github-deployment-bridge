// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package metadata

// ApplyPayloadExtras adds optional resolved annotation fields to a GitHub
// Deployment payload. Keys use the bridge's camelCase payload convention.
func ApplyPayloadExtras(payload map[string]any, resolved Resolved) {
	setIfNonempty(payload, "team", resolved.Team)
	setIfNonempty(payload, "service", resolved.Service)
	setIfNonempty(payload, "component", resolved.Component)
	setIfNonempty(payload, "slackChannel", resolved.SlackChannel)
	setIfNonempty(payload, "owner", resolved.Owner)
	setIfNonempty(payload, "release", resolved.Release)
	setIfNonempty(payload, "tag", resolved.Tag)
}

func setIfNonempty(payload map[string]any, key, value string) {
	if value != "" {
		payload[key] = value
	}
}
