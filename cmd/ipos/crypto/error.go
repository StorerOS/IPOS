package crypto

import (
	"fmt"
)

type Error struct {
	err error
}

func Errorf(format string, a ...interface{}) error {
	return Error{err: fmt.Errorf(format, a...)}
}

func (e Error) Unwrap() error { return e.err }

func (e Error) Error() string {
	if e.err == nil {
		return "crypto: cause <nil>"
	}
	return e.err.Error()
}

var (
	ErrInvalidEncryptionMethod = Errorf("The encryption method is not supported")

	ErrInvalidCustomerAlgorithm = Errorf("The SSE-C algorithm is not supported")

	ErrMissingCustomerKey = Errorf("The SSE-C request is missing the customer key")

	ErrMissingCustomerKeyMD5 = Errorf("The SSE-C request is missing the customer key MD5")

	ErrInvalidCustomerKey = Errorf("The SSE-C client key is invalid")

	ErrSecretKeyMismatch = Errorf("The secret key does not match the secret key used during upload")

	ErrCustomerKeyMD5Mismatch       = Errorf("The provided SSE-C key MD5 does not match the computed MD5 of the SSE-C key")
	ErrIncompatibleEncryptionMethod = Errorf("Server side encryption specified with both SSE-C and SSE-S3 headers")
)

var (
	errMissingInternalIV            = Errorf("The object metadata is missing the internal encryption IV")
	errMissingInternalSealAlgorithm = Errorf("The object metadata is missing the internal seal algorithm")

	errInvalidInternalIV            = Errorf("The internal encryption IV is malformed")
	errInvalidInternalSealAlgorithm = Errorf("The internal seal algorithm is invalid and not supported")

	errMissingUpdatedKey = Errorf("The key update returned no error but also no sealed key")
)

var (
	errOutOfEntropy = Errorf("Unable to read enough randomness from the system")
)
