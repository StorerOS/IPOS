// +build !windows

package cli

const (
	defaultPrompt             = "$"
	defaultEnvSetCmd          = "export"
	defaultAssignmentOperator = "="
	defaultDisableHistory     = "$ set +o history"
	defaultEnableHistory      = "$ set -o history"
)
