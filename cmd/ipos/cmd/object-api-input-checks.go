package cmd

import (
	"context"

	"github.com/storeros/ipos/cmd/ipos/logger"
)

func checkListObjsArgs(ctx context.Context, bucket, prefix, marker string, obj ObjectLayer) error {
	if err := checkBucketExist(ctx, bucket, obj); err != nil {
		return err
	}
	if !IsValidObjectPrefix(prefix) {
		logger.LogIf(ctx, ObjectNameInvalid{
			Bucket: bucket,
			Object: prefix,
		})
		return ObjectNameInvalid{
			Bucket: bucket,
			Object: prefix,
		}
	}
	if marker != "" && !HasPrefix(marker, prefix) {
		logger.LogIf(ctx, InvalidMarkerPrefixCombination{
			Marker: marker,
			Prefix: prefix,
		})
		return InvalidMarkerPrefixCombination{
			Marker: marker,
			Prefix: prefix,
		}
	}
	return nil
}

func checkBucketExist(ctx context.Context, bucket string, obj ObjectLayer) error {
	_, err := obj.GetBucketInfo(ctx, bucket)
	if err != nil {
		return err
	}
	return nil
}
