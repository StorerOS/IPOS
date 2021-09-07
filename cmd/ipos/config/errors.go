package config

var (
	ErrPortAlreadyInUse = newErrFn(
		"Port is already in use",
		"Please ensure no other program uses the same address/port",
		"",
	)

	ErrPortAccess = newErrFn(
		"Unable to use specified port",
		"Please ensure IPOS binary has 'cap_net_bind_service=+ep' permissions",
		`Use 'sudo setcap cap_net_bind_service=+ep /path/to/ipos' to provide sufficient permissions`,
	)

	ErrNoPermissionsToAccessDirFiles = newErrFn(
		"Missing permissions to access the specified path",
		"Please ensure the specified path can be accessed",
		"",
	)

	ErrUnexpectedDataContent = newErrFn(
		"Unexpected data content",
		"Please contact IPOS at https://ipos.storeros.com",
		"",
	)
)
