package cmd

import (
	"context"
	"sort"
	"strings"
)

type TreeWalkResult struct {
	entry string
	end   bool
}

func filterMatchingPrefix(entries []string, prefixEntry string) []string {
	start := 0
	end := len(entries)
	for {
		if start == end {
			break
		}
		if HasPrefix(entries[start], prefixEntry) {
			break
		}
		start++
	}
	for {
		if start == end {
			break
		}
		if HasPrefix(entries[end-1], prefixEntry) {
			break
		}
		end--
	}
	sort.Strings(entries[start:end])
	return entries[start:end]
}

type ListDirFunc func(bucket, prefixDir, prefixEntry string) (emptyDir bool, entries []string)

func doTreeWalk(ctx context.Context, bucket, prefixDir, entryPrefixMatch, marker string, recursive bool, listDir ListDirFunc, resultCh chan TreeWalkResult, endWalkCh <-chan struct{}, isEnd bool) (emptyDir bool, treeErr error) {

	var markerBase, markerDir string
	if marker != "" {
		markerSplit := strings.SplitN(marker, SlashSeparator, 2)
		markerDir = markerSplit[0]
		if len(markerSplit) == 2 {
			markerDir += SlashSeparator
			markerBase = markerSplit[1]
		}
	}

	emptyDir, entries := listDir(bucket, prefixDir, entryPrefixMatch)
	if emptyDir {
		return true, nil
	}

	idx := sort.Search(len(entries), func(i int) bool {
		return entries[i] >= markerDir
	})
	entries = entries[idx:]
	if len(entries) == 0 {
		return false, nil
	}

	for i, entry := range entries {
		pentry := pathJoin(prefixDir, entry)
		isDir := HasSuffix(pentry, SlashSeparator)

		if i == 0 && markerDir == entry {
			if !recursive {
				continue
			}
			if recursive && !isDir {

				continue
			}
		}
		if recursive && isDir {
			markerArg := ""
			if entry == markerDir {
				markerArg = markerBase
			}
			prefixMatch := ""
			markIsEnd := i == len(entries)-1 && isEnd
			emptyDir, err := doTreeWalk(ctx, bucket, pentry, prefixMatch, markerArg, recursive,
				listDir, resultCh, endWalkCh, markIsEnd)
			if err != nil {
				return false, err
			}

			if !emptyDir {
				continue
			}
		}

		isEOF := ((i == len(entries)-1) && isEnd)
		select {
		case <-endWalkCh:
			return false, errWalkAbort
		case resultCh <- TreeWalkResult{entry: pentry, end: isEOF}:
		}
	}

	return false, nil
}

func startTreeWalk(ctx context.Context, bucket, prefix, marker string, recursive bool, listDir ListDirFunc, endWalkCh <-chan struct{}) chan TreeWalkResult {

	resultCh := make(chan TreeWalkResult, maxObjectList)
	entryPrefixMatch := prefix
	prefixDir := ""
	lastIndex := strings.LastIndex(prefix, SlashSeparator)
	if lastIndex != -1 {
		entryPrefixMatch = prefix[lastIndex+1:]
		prefixDir = prefix[:lastIndex+1]
	}
	marker = strings.TrimPrefix(marker, prefixDir)
	go func() {
		isEnd := true
		doTreeWalk(ctx, bucket, prefixDir, entryPrefixMatch, marker, recursive, listDir, resultCh, endWalkCh, isEnd)
		close(resultCh)
	}()
	return resultCh
}
