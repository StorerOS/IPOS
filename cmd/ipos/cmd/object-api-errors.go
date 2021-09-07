package cmd

import (
	"errors"
	"fmt"
	"io"
	"path"
)

func toObjectErr(err error, params ...string) error {
	switch err {
	case errDataTooLarge:
		if len(params) >= 2 {
			err = ObjectTooLarge{
				Bucket: params[0],
				Object: params[1],
			}
		}
	case errDataTooSmall:
		if len(params) >= 2 {
			err = ObjectTooSmall{
				Bucket: params[0],
				Object: params[1],
			}
		}
	case io.ErrUnexpectedEOF, io.ErrShortWrite:
		err = IncompleteBody{}
	}
	return err
}

type SignatureDoesNotMatch struct{}

func (e SignatureDoesNotMatch) Error() string {
	return "The request signature we calculated does not match the signature you provided. Check your key and signing method."
}

type StorageFull struct{}

func (e StorageFull) Error() string {
	return "Storage reached its minimum free disk threshold."
}

type SlowDown struct{}

func (e SlowDown) Error() string {
	return "Please reduce your request rate"
}

type InsufficientReadQuorum struct{}

func (e InsufficientReadQuorum) Error() string {
	return "Storage resources are insufficient for the read operation."
}

type InsufficientWriteQuorum struct{}

func (e InsufficientWriteQuorum) Error() string {
	return "Storage resources are insufficient for the write operation."
}

type GenericError struct {
	Bucket string
	Object string
}

type BucketNotFound GenericError

func (e BucketNotFound) Error() string {
	return "Bucket not found: " + e.Bucket
}

type BucketAlreadyExists GenericError

func (e BucketAlreadyExists) Error() string {
	return "The requested bucket name is not available. The bucket namespace is shared by all users of the system. Please select a different name and try again."
}

type BucketAlreadyOwnedByYou GenericError

func (e BucketAlreadyOwnedByYou) Error() string {
	return "Bucket already owned by you: " + e.Bucket
}

type BucketNotEmpty GenericError

func (e BucketNotEmpty) Error() string {
	return "Bucket not empty: " + e.Bucket
}

type ObjectNotFound GenericError

func (e ObjectNotFound) Error() string {
	return "Object not found: " + e.Bucket + "#" + e.Object
}

type ObjectAlreadyExists GenericError

func (e ObjectAlreadyExists) Error() string {
	return "Object: " + e.Bucket + "#" + e.Object + " already exists"
}

type ObjectExistsAsDirectory GenericError

func (e ObjectExistsAsDirectory) Error() string {
	return "Object exists on : " + e.Bucket + " as directory " + e.Object
}

type PrefixAccessDenied GenericError

func (e PrefixAccessDenied) Error() string {
	return "Prefix access is denied: " + e.Bucket + SlashSeparator + e.Object
}

type ParentIsObject GenericError

func (e ParentIsObject) Error() string {
	return "Parent is object " + e.Bucket + SlashSeparator + path.Dir(e.Object)
}

type BucketExists GenericError

func (e BucketExists) Error() string {
	return "Bucket exists: " + e.Bucket
}

type UnsupportedDelimiter struct {
	Delimiter string
}

func (e UnsupportedDelimiter) Error() string {
	return fmt.Sprintf("delimiter '%s' is not supported. Only '/' is supported", e.Delimiter)
}

type InvalidUploadIDKeyCombination struct {
	UploadIDMarker, KeyMarker string
}

func (e InvalidUploadIDKeyCombination) Error() string {
	return fmt.Sprintf("Invalid combination of uploadID marker '%s' and marker '%s'", e.UploadIDMarker, e.KeyMarker)
}

type InvalidMarkerPrefixCombination struct {
	Marker, Prefix string
}

func (e InvalidMarkerPrefixCombination) Error() string {
	return fmt.Sprintf("Invalid combination of marker '%s' and prefix '%s'", e.Marker, e.Prefix)
}

type BucketPolicyNotFound GenericError

func (e BucketPolicyNotFound) Error() string {
	return "No bucket policy found for bucket: " + e.Bucket
}

type BucketLifecycleNotFound GenericError

func (e BucketLifecycleNotFound) Error() string {
	return "No bucket life cycle found for bucket : " + e.Bucket
}

type BucketSSEConfigNotFound GenericError

func (e BucketSSEConfigNotFound) Error() string {
	return "No bucket encryption found for bucket: " + e.Bucket
}

type BucketNameInvalid GenericError

func (e BucketNameInvalid) Error() string {
	return "Bucket name invalid: " + e.Bucket
}

type ObjectNameInvalid GenericError

type ObjectNameTooLong GenericError

type ObjectNamePrefixAsSlash GenericError

func (e ObjectNameInvalid) Error() string {
	return "Object name invalid: " + e.Bucket + "#" + e.Object
}

func (e ObjectNameTooLong) Error() string {
	return "Object name too long: " + e.Bucket + "#" + e.Object
}

func (e ObjectNamePrefixAsSlash) Error() string {
	return "Object name contains forward slash as pefix: " + e.Bucket + "#" + e.Object
}

type AllAccessDisabled GenericError

func (e AllAccessDisabled) Error() string {
	return "All access to this object has been disabled"
}

type IncompleteBody GenericError

func (e IncompleteBody) Error() string {
	return e.Bucket + "#" + e.Object + "has incomplete body"
}

type InvalidRange struct {
	OffsetBegin  int64
	OffsetEnd    int64
	ResourceSize int64
}

func (e InvalidRange) Error() string {
	return fmt.Sprintf("The requested range \"bytes %d-%d/%d\" is not satisfiable.", e.OffsetBegin, e.OffsetEnd, e.ResourceSize)
}

type ObjectTooLarge GenericError

func (e ObjectTooLarge) Error() string {
	return "size of the object greater than what is allowed(5G)"
}

type ObjectTooSmall GenericError

func (e ObjectTooSmall) Error() string {
	return "size of the object less than what is expected"
}

type OperationTimedOut struct {
}

func (e OperationTimedOut) Error() string {
	return "Operation timed out"
}

type MalformedUploadID struct {
	UploadID string
}

func (e MalformedUploadID) Error() string {
	return "Malformed upload id " + e.UploadID
}

type InvalidUploadID struct {
	Bucket   string
	Object   string
	UploadID string
}

func (e InvalidUploadID) Error() string {
	return "Invalid upload id " + e.UploadID
}

type InvalidPart struct {
	PartNumber int
	ExpETag    string
	GotETag    string
}

func (e InvalidPart) Error() string {
	return fmt.Sprintf("Specified part could not be found. PartNumber %d, Expected %s, got %s",
		e.PartNumber, e.ExpETag, e.GotETag)
}

type PartTooSmall struct {
	PartSize   int64
	PartNumber int
	PartETag   string
}

func (e PartTooSmall) Error() string {
	return fmt.Sprintf("Part size for %d should be at least 5MB", e.PartNumber)
}

type PartTooBig struct{}

func (e PartTooBig) Error() string {
	return "Part size bigger than the allowed limit"
}

type InvalidETag struct{}

func (e InvalidETag) Error() string {
	return "etag of the object has changed"
}

type NotImplemented struct{}

func (e NotImplemented) Error() string {
	return "Not Implemented"
}

type UnsupportedMetadata struct{}

func (e UnsupportedMetadata) Error() string {
	return "Unsupported headers in Metadata"
}

type BackendDown struct{}

func (e BackendDown) Error() string {
	return "Backend down"
}

func isErrBucketNotFound(err error) bool {
	var bkNotFound BucketNotFound
	return errors.As(err, &bkNotFound)
}

func isErrObjectNotFound(err error) bool {
	var objNotFound ObjectNotFound
	return errors.As(err, &objNotFound)
}

type PreConditionFailed struct{}

func (e PreConditionFailed) Error() string {
	return "At least one of the pre-conditions you specified did not hold"
}

func isErrPreconditionFailed(err error) bool {
	_, ok := err.(PreConditionFailed)
	return ok
}
