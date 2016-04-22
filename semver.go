// Copyright 2014 The Semver Package Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package semver contains types and functions for
// parsing of Versions and (Version-)Ranges.
package semver

import (
	"errors"
	"strconv"
)

// Errors that are thrown when translating from a string.
var (
	ErrInvalidVersionString = errors.New("Given string does not resemble a Version")
	ErrTooMuchColumns       = errors.New("Version consists of too much columns")
)

// alpha = -4, beta = -3, pre = -2, rc = -1, common = 0, revision = 1, patch = 2
const (
	alpha = iota - 4
	beta
	pre
	rc
	common
	revision
	patch
)

const (
	idxReleaseType   = 4
	idxRelease       = 5
	idxSpecifierType = 9
	idxSpecifier     = 10
)

var releaseDesc = map[int]string{
	alpha:    "alpha",
	beta:     "beta",
	pre:      "pre",
	rc:       "rc",
	revision: "r",
	patch:    "p",
}

var releaseValue = map[string]int{
	"alpha": alpha,
	"beta":  beta,
	"pre":   pre,
	"":      pre,
	"rc":    rc,
	"r":     revision,
	"p":     patch,
}

// Version represents a version:
// Columns consisting of up to four unsigned integers (1.2.4.99)
// optionally further divided into 'release' and 'specifier' (1.2-634.0-99.8).
type Version struct {
	// 0–3: version, 4: releaseType, 5–8: releaseVer, 9: releaseSpecifier, 10–14: specifier
	version [14]int
	build   int
}

// NewVersion translates the given string, which must be free of whitespace,
// into a single Version.
func NewVersion(str string) (*Version, error) {
	ver := &Version{}
	err := ver.Parse(str)
	return ver, err
}

// Parse reads a string into the given version, overwriting any existing values.
func (t *Version) Parse(str string) error {
	var fromIdx, fromLen, fieldNum int
	var isAlpha bool
	var strlen = len(str)

	for idx, r := range str {
		// consume
		if isAlpha { // a-z
			if 'a' <= r && r <= 'z' {
				fromLen++
				if idx+1 < strlen {
					continue
				}
			}
		} else { // numbers
			if '0' <= r && r <= '9' {
				fromLen++
				if idx+1 < strlen {
					continue
				}
			}
		}

		// convert
		if isAlpha {
			switch {
			case fieldNum <= idxReleaseType:
				fieldNum = idxReleaseType
			case fieldNum <= idxSpecifierType:
				fieldNum = idxSpecifierType
			default:
				return ErrInvalidVersionString
			}

			typ, known := releaseValue[str[fromIdx:fromIdx+fromLen]]
			if !known {
				return ErrInvalidVersionString
			}
			t.version[fieldNum] = typ
		} else {
			if fieldNum == idxReleaseType || fieldNum == idxSpecifierType {
				return ErrTooMuchColumns
			}

			n, err := strconv.Atoi(str[fromIdx : fromIdx+fromLen])
			if err != nil {
				return err
			}
			t.version[fieldNum] = n
		}
		fieldNum++
		fromLen = 0

		switch r {
		case '.':
			fromIdx = idx + 1
			isAlpha = false
		case '-', '_':
			fromIdx = idx + 1
			if strlen < fromIdx {
				return ErrInvalidVersionString
			}
			isAlpha = 'a' <= str[fromIdx] && str[fromIdx] <= 'z'
			switch {
			case fieldNum <= idxReleaseType:
				fieldNum = idxReleaseType
			case fieldNum <= idxSpecifierType:
				fieldNum = idxSpecifierType
			default:
				return ErrInvalidVersionString
			}
			if !isAlpha {
				fieldNum++
			}
		case '+': // special case: build
			if strlen < idx+7 || str[idx:idx+6] != "+build" {
				return errors.New("Version has no suffix +build and numbers")
			}
			n, err := strconv.Atoi(str[idx+6:])
			if err != nil {
				return err
			}
			t.build = n
			return nil
		default:
			fromIdx = idx
			isAlpha = 'a' <= r && r <= 'z'
			fromLen = 1
		}

		if fieldNum > 14 {
			return errors.New("Version is too long")
		}
	}

	return nil
}

// signDelta returns the signum of the difference,
// which' precision can be limited by 'cuttofIdx'.
func signDelta(a, b [14]int, cutoffIdx int) int8 {
	//fmt.Println(a, b)
	for i := range a {
		if i >= cutoffIdx {
			return 0
		}
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// Compare computes the difference between two Versions and returns its signum.
//
//   1  if a > b
//   0  if a == b
//   -1 if a < b
//
// The 'build' is not compared.
func Compare(a, b Version) int {
	return int(signDelta(a.version, b.version, 14))
}

// Less is a convenience function for sorting.
func (t *Version) Less(o *Version) bool {
	sd := signDelta(t.version, o.version, 15)
	return sd < 0 || (sd == 0 && t.build < o.build)
}

// limitedLess compares two Versions
// with a precision limited to version, (pre-)release type and (pre-)release version.
//
// Commutative.
func (t *Version) limitedLess(o *Version) bool {
	return signDelta(t.version, o.version, idxSpecifierType) < 0
}

// LimitedEqual returns true of two versions share the same prefix,
// which is the "actual version", (pre-)release type, and (pre-)release version.
// The exception are patch-levels, which are always equal.
//
// Use this, for example, to tell a beta from a regular version;
// or to accept a patched version as regular version.
func (t *Version) LimitedEqual(o *Version) bool {
	if t.version[idxReleaseType] == common && o.version[idxReleaseType] > common {
		return t.sharesPrefixWith(o)
	}
	return signDelta(t.version, o.version, idxSpecifierType) == 0
}

// IsAPreRelease is used to discriminate pre-releases.
func (t *Version) IsAPreRelease() bool {
	return t.version[idxReleaseType] < common
}

// sharesPrefixWith compares two Versions with a fixed limited precision.
//
// A 'prefix' is the major, minor, patch and revision number.
// For example: 1.2.3.4…
func (t *Version) sharesPrefixWith(o *Version) bool {
	return signDelta(t.version, o.version, idxReleaseType) == 0
}
