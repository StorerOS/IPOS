package crypto

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/storeros/ipos/cmd/ipos/logger"
)

func IsMultiPart(metadata map[string]string) bool {
	if _, ok := metadata[SSEMultipart]; ok {
		return true
	}
	return false
}

func RemoveSensitiveEntries(metadata map[string]string) {
	delete(metadata, SSECKey)
	delete(metadata, SSECopyKey)
}

func RemoveSSEHeaders(metadata map[string]string) {
	delete(metadata, SSEHeader)
	delete(metadata, SSECKeyMD5)
	delete(metadata, SSECAlgorithm)
}

func RemoveInternalEntries(metadata map[string]string) {
	delete(metadata, SSEMultipart)
	delete(metadata, SSEIV)
	delete(metadata, SSESealAlgorithm)
	delete(metadata, SSECSealedKey)
	delete(metadata, S3SealedKey)
	delete(metadata, S3KMSKeyID)
	delete(metadata, S3KMSSealedKey)
}

func IsEncrypted(metadata map[string]string) bool {
	if _, ok := metadata[SSEIV]; ok {
		return true
	}
	if _, ok := metadata[SSESealAlgorithm]; ok {
		return true
	}
	if IsMultiPart(metadata) {
		return true
	}
	if S3.IsEncrypted(metadata) {
		return true
	}
	if SSEC.IsEncrypted(metadata) {
		return true
	}
	return false
}

func (s3) IsEncrypted(metadata map[string]string) bool {
	if _, ok := metadata[S3SealedKey]; ok {
		return true
	}
	if _, ok := metadata[S3KMSKeyID]; ok {
		return true
	}
	if _, ok := metadata[S3KMSSealedKey]; ok {
		return true
	}
	return false
}

func (ssec) IsEncrypted(metadata map[string]string) bool {
	if _, ok := metadata[SSECSealedKey]; ok {
		return true
	}
	return false
}

func CreateMultipartMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata[SSEMultipart] = ""
	return metadata
}

func (s3) CreateMetadata(metadata map[string]string, keyID string, kmsKey []byte, sealedKey SealedKey) map[string]string {
	if sealedKey.Algorithm != SealAlgorithm {
		logger.CriticalIf(context.Background(), Errorf("The seal algorithm '%s' is invalid for SSE-S3", sealedKey.Algorithm))
	}

	if keyID == "" && len(kmsKey) != 0 {
		logger.CriticalIf(context.Background(), errors.New("The key ID must not be empty if a KMS data key is present"))
	}
	if keyID != "" && len(kmsKey) == 0 {
		logger.CriticalIf(context.Background(), errors.New("The KMS data key must not be empty if a key ID is present"))
	}

	if metadata == nil {
		metadata = map[string]string{}
	}

	metadata[SSESealAlgorithm] = sealedKey.Algorithm
	metadata[SSEIV] = base64.StdEncoding.EncodeToString(sealedKey.IV[:])
	metadata[S3SealedKey] = base64.StdEncoding.EncodeToString(sealedKey.Key[:])
	if len(kmsKey) > 0 && keyID != "" {
		metadata[S3KMSKeyID] = keyID
		metadata[S3KMSSealedKey] = base64.StdEncoding.EncodeToString(kmsKey)
	}
	return metadata
}

func (s3) ParseMetadata(metadata map[string]string) (keyID string, kmsKey []byte, sealedKey SealedKey, err error) {
	b64IV, ok := metadata[SSEIV]
	if !ok {
		return keyID, kmsKey, sealedKey, errMissingInternalIV
	}
	algorithm, ok := metadata[SSESealAlgorithm]
	if !ok {
		return keyID, kmsKey, sealedKey, errMissingInternalSealAlgorithm
	}
	b64SealedKey, ok := metadata[S3SealedKey]
	if !ok {
		return keyID, kmsKey, sealedKey, Errorf("The object metadata is missing the internal sealed key for SSE-S3")
	}

	keyID, idPresent := metadata[S3KMSKeyID]
	b64KMSSealedKey, kmsKeyPresent := metadata[S3KMSSealedKey]
	if !idPresent && kmsKeyPresent {
		return keyID, kmsKey, sealedKey, Errorf("The object metadata is missing the internal KMS key-ID for SSE-S3")
	}
	if idPresent && !kmsKeyPresent {
		return keyID, kmsKey, sealedKey, Errorf("The object metadata is missing the internal sealed KMS data key for SSE-S3")
	}

	iv, err := base64.StdEncoding.DecodeString(b64IV)
	if err != nil || len(iv) != 32 {
		return keyID, kmsKey, sealedKey, errInvalidInternalIV
	}
	if algorithm != SealAlgorithm {
		return keyID, kmsKey, sealedKey, errInvalidInternalSealAlgorithm
	}
	encryptedKey, err := base64.StdEncoding.DecodeString(b64SealedKey)
	if err != nil || len(encryptedKey) != 64 {
		return keyID, kmsKey, sealedKey, Errorf("The internal sealed key for SSE-S3 is invalid")
	}
	if idPresent && kmsKeyPresent {
		kmsKey, err = base64.StdEncoding.DecodeString(b64KMSSealedKey)
		if err != nil {
			return keyID, kmsKey, sealedKey, Errorf("The internal sealed KMS data key for SSE-S3 is invalid")
		}
	}

	sealedKey.Algorithm = algorithm
	copy(sealedKey.IV[:], iv)
	copy(sealedKey.Key[:], encryptedKey)
	return keyID, kmsKey, sealedKey, nil
}

func (ssec) CreateMetadata(metadata map[string]string, sealedKey SealedKey) map[string]string {
	if sealedKey.Algorithm != SealAlgorithm {
		logger.CriticalIf(context.Background(), Errorf("The seal algorithm '%s' is invalid for SSE-C", sealedKey.Algorithm))
	}

	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata[SSESealAlgorithm] = SealAlgorithm
	metadata[SSEIV] = base64.StdEncoding.EncodeToString(sealedKey.IV[:])
	metadata[SSECSealedKey] = base64.StdEncoding.EncodeToString(sealedKey.Key[:])
	return metadata
}

func (ssec) ParseMetadata(metadata map[string]string) (sealedKey SealedKey, err error) {
	b64IV, ok := metadata[SSEIV]
	if !ok {
		return sealedKey, errMissingInternalIV
	}
	algorithm, ok := metadata[SSESealAlgorithm]
	if !ok {
		return sealedKey, errMissingInternalSealAlgorithm
	}
	b64SealedKey, ok := metadata[SSECSealedKey]
	if !ok {
		return sealedKey, Errorf("The object metadata is missing the internal sealed key for SSE-C")
	}

	iv, err := base64.StdEncoding.DecodeString(b64IV)
	if err != nil || len(iv) != 32 {
		return sealedKey, errInvalidInternalIV
	}
	if algorithm != SealAlgorithm && algorithm != InsecureSealAlgorithm {
		return sealedKey, errInvalidInternalSealAlgorithm
	}
	encryptedKey, err := base64.StdEncoding.DecodeString(b64SealedKey)
	if err != nil || len(encryptedKey) != 64 {
		return sealedKey, Errorf("The internal sealed key for SSE-C is invalid")
	}

	sealedKey.Algorithm = algorithm
	copy(sealedKey.IV[:], iv)
	copy(sealedKey.Key[:], encryptedKey)
	return sealedKey, nil
}

func IsETagSealed(etag []byte) bool { return len(etag) > 16 }
