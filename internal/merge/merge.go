// Package merge implements deep-merge of JSON settings maps.
package merge

import (
	"reflect"
	"sort"
)

// Conflict represents a key present in both master and local where values differ.
type Conflict struct {
	Key         string
	MasterValue any
	LocalValue  any
}

// Result holds the outcome of a merge operation.
type Result struct {
	Merged    map[string]any
	Conflicts []Conflict
	Forced    []string // keys where master overwrote local due to force
	Added     []string // keys from master not in local
	Matching  []string // keys present in both with identical values
	LocalOnly []string // keys in local not present in master
}

// Merge combines master into local. Keys already present in local are kept
// unless force is true, in which case conflicting keys use the master value.
// Nested objects are recursively merged. Keys with identical values are counted
// as matching. Keys with differing values are recorded as conflicts (or forced
// if force is true). Keys present only in local are recorded for awareness.
func Merge(master, local map[string]any, force bool) Result {
	result := Result{
		Merged: make(map[string]any, len(local)),
	}

	// Copy all local keys first.
	for k, v := range local {
		result.Merged[k] = v
	}

	mergeInto(result.Merged, master, local, "", force, &result)

	sort.Strings(result.Added)
	sort.Strings(result.Matching)
	sort.Strings(result.LocalOnly)
	sort.Strings(result.Forced)
	sort.Slice(result.Conflicts, func(i, j int) bool {
		return result.Conflicts[i].Key < result.Conflicts[j].Key
	})

	return result
}

// mergeInto recursively merges src into dst, tracking additions, matches, conflicts, and local-only keys.
func mergeInto(dst, src, localSrc map[string]any, prefix string, force bool, result *Result) {
	for k, srcVal := range src {
		key := qualifiedKey(prefix, k)

		dstVal, exists := dst[k]
		if !exists {
			dst[k] = srcVal
			result.Added = append(result.Added, key)
			continue
		}

		// Both exist — recurse if both are objects, otherwise compare values.
		srcMap, srcIsMap := srcVal.(map[string]any)
		dstMap, dstIsMap := dstVal.(map[string]any)

		if srcIsMap && dstIsMap {
			var localSubMap map[string]any
			if localSrc != nil {
				localSubMap, _ = localSrc[k].(map[string]any)
			}
			mergeInto(dstMap, srcMap, localSubMap, key, force, result)
			dst[k] = dstMap
			continue
		}

		if reflect.DeepEqual(srcVal, dstVal) {
			result.Matching = append(result.Matching, key)
			continue
		}

		if force {
			dst[k] = srcVal
			result.Forced = append(result.Forced, key)
			continue
		}

		result.Conflicts = append(result.Conflicts, Conflict{
			Key:         key,
			MasterValue: srcVal,
			LocalValue:  dstVal,
		})
	}

	// Find keys in local not present in master.
	for k := range dst {
		key := qualifiedKey(prefix, k)
		if _, inMaster := src[k]; !inMaster {
			// Only record if this key existed in the original local (not added from master).
			if localSrc != nil {
				if _, inLocal := localSrc[k]; inLocal {
					result.LocalOnly = append(result.LocalOnly, key)
				}
			}
		}
	}
}

func qualifiedKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
