package cmd

import (
	"context"
	"strings"
	"sync"

	humanize "github.com/dustin/go-humanize"
)

const (
	blockSizeV1 = 10 * humanize.MiByte

	readSizeV1 = 1 * humanize.MiByte

	bucketMetaPrefix = "buckets"

	emptyETag = "d41d8cd98f00b204e9800998ecf8427e"
)

var globalObjLayerMutex *sync.RWMutex

var globalObjectAPI ObjectLayer

func init() {
	globalObjLayerMutex = &sync.RWMutex{}
}

func listObjectsNonSlash(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int, tpool *TreeWalkPool, listDir ListDirFunc, getObjInfo func(context.Context, string, string) (ObjectInfo, error), getObjectInfoDirs ...func(context.Context, string, string) (ObjectInfo, error)) (loi ListObjectsInfo, err error) {
	endWalkCh := make(chan struct{})
	defer close(endWalkCh)
	recursive := true
	walkResultCh := startTreeWalk(ctx, bucket, prefix, "", recursive, listDir, endWalkCh)

	var objInfos []ObjectInfo
	var eof bool
	var prevPrefix string

	for {
		if len(objInfos) == maxKeys {
			break
		}
		result, ok := <-walkResultCh
		if !ok {
			eof = true
			break
		}

		var objInfo ObjectInfo
		var err error

		index := strings.Index(strings.TrimPrefix(result.entry, prefix), delimiter)
		if index == -1 {
			objInfo, err = getObjInfo(ctx, bucket, result.entry)
			if err != nil {
				if IsErrIgnored(err, []error{
					errFileNotFound,
				}...) {
					continue
				}
				return loi, toObjectErr(err, bucket, prefix)
			}
		} else {
			index = len(prefix) + index + len(delimiter)
			currPrefix := result.entry[:index]
			if currPrefix == prevPrefix {
				continue
			}
			prevPrefix = currPrefix

			objInfo = ObjectInfo{
				Bucket: bucket,
				Name:   currPrefix,
				IsDir:  true,
			}
		}

		if objInfo.Name <= marker {
			continue
		}

		objInfos = append(objInfos, objInfo)
		if result.end {
			eof = true
			break
		}
	}

	result := ListObjectsInfo{}
	for _, objInfo := range objInfos {
		if objInfo.IsDir {
			result.Prefixes = append(result.Prefixes, objInfo.Name)
			continue
		}
		result.Objects = append(result.Objects, objInfo)
	}

	if !eof {
		result.IsTruncated = true
		if len(objInfos) > 0 {
			result.NextMarker = objInfos[len(objInfos)-1].Name
		}
	}

	return result, nil
}

func listObjects(ctx context.Context, obj ObjectLayer, bucket, prefix, marker, delimiter string, maxKeys int, tpool *TreeWalkPool, listDir ListDirFunc, getObjInfo func(context.Context, string, string) (ObjectInfo, error), getObjectInfoDirs ...func(context.Context, string, string) (ObjectInfo, error)) (loi ListObjectsInfo, err error) {
	if delimiter != SlashSeparator && delimiter != "" {
		return listObjectsNonSlash(ctx, bucket, prefix, marker, delimiter, maxKeys, tpool, listDir, getObjInfo, getObjectInfoDirs...)
	}

	if err := checkListObjsArgs(ctx, bucket, prefix, marker, obj); err != nil {
		return loi, err
	}

	if marker != "" {
		if !HasPrefix(marker, prefix) {
			return loi, nil
		}
	}

	if maxKeys == 0 {
		return loi, nil
	}

	if delimiter == SlashSeparator && prefix == SlashSeparator {
		return loi, nil
	}

	if maxKeys < 0 || maxKeys > maxObjectList {
		maxKeys = maxObjectList
	}

	recursive := true
	if delimiter == SlashSeparator {
		recursive = false
	}

	walkResultCh, endWalkCh := tpool.Release(listParams{bucket, recursive, marker, prefix})
	if walkResultCh == nil {
		endWalkCh = make(chan struct{})
		walkResultCh = startTreeWalk(ctx, bucket, prefix, marker, recursive, listDir, endWalkCh)
	}

	var objInfos []ObjectInfo
	var eof bool
	var nextMarker string

	for i := 0; i < maxKeys; {
		walkResult, ok := <-walkResultCh
		if !ok {
			eof = true
			break
		}

		var objInfo ObjectInfo
		var err error
		if HasSuffix(walkResult.entry, SlashSeparator) {
			for _, getObjectInfoDir := range getObjectInfoDirs {
				objInfo, err = getObjectInfoDir(ctx, bucket, walkResult.entry)
				if err == nil {
					break
				}
				if err == errFileNotFound {
					err = nil
					objInfo = ObjectInfo{
						Bucket: bucket,
						Name:   walkResult.entry,
						IsDir:  true,
					}
				}
			}
		} else {
			objInfo, err = getObjInfo(ctx, bucket, walkResult.entry)
		}
		if err != nil {
			if IsErrIgnored(err, []error{
				errFileNotFound,
			}...) {
				continue
			}
			return loi, toObjectErr(err, bucket, prefix)
		}
		nextMarker = objInfo.Name
		objInfos = append(objInfos, objInfo)
		if walkResult.end {
			eof = true
			break
		}
		i++
	}

	params := listParams{bucket, recursive, nextMarker, prefix}
	if !eof {
		tpool.Set(params, walkResultCh, endWalkCh)
	}

	result := ListObjectsInfo{}
	for _, objInfo := range objInfos {
		if objInfo.IsDir && delimiter == SlashSeparator {
			result.Prefixes = append(result.Prefixes, objInfo.Name)
			continue
		}
		result.Objects = append(result.Objects, objInfo)
	}

	if !eof {
		result.IsTruncated = true
		if len(objInfos) > 0 {
			result.NextMarker = objInfos[len(objInfos)-1].Name
		}
	}

	return result, nil
}
