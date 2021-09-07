package cmd

import (
	"errors"
)

var errInvalidArgument = errors.New("Invalid arguments specified")

var errMethodNotAllowed = errors.New("Method not allowed")

var errSignatureMismatch = errors.New("Signature does not match")

var errSizeUnexpected = errors.New("Data size larger than expected")

var errSizeUnspecified = errors.New("Data size is unspecified")

var errDataTooLarge = errors.New("Object size larger than allowed limit")

var errDataTooSmall = errors.New("Object size smaller than expected")

var errServerNotInitialized = errors.New("Server not initialized, please try again")

var errRPCAPIVersionUnsupported = errors.New("Unsupported rpc API version")

var errServerTimeMismatch = errors.New("Server times are too far apart")

var errInvalidBucketName = errors.New("The specified bucket is not valid")

var errInvalidRange = errors.New("Invalid range")

var errInvalidRangeSource = errors.New("Range specified exceeds source object size")

var errNotFirstDisk = errors.New("Not first disk")

var errFirstDiskWait = errors.New("Waiting on other disks")

var errBucketAlreadyExists = errors.New("Your previous request to create the named bucket succeeded and you already own it")

var errInvalidDecompressedSize = errors.New("Invalid Decompressed Size")

var errNoSuchUser = errors.New("Specified user does not exist")

var errNoSuchServiceAccount = errors.New("Specified service account does not exist")

var errNoSuchGroup = errors.New("Specified group does not exist")

var errGroupNotEmpty = errors.New("Specified group is not empty - cannot remove it")

var errNoSuchPolicy = errors.New("Specified canned policy does not exist")

var errIAMActionNotAllowed = errors.New("Specified IAM action is not allowed under the current configuration")

var errAccessDenied = errors.New("Do not have enough permissions to access this resource")
