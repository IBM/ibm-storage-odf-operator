/**
 * Copyright contributors to the ibm-storage-odf-operator project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
// controllers/util/safeio.go
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindModuleRoot walks up from the current directory until it finds go.mod.
// It returns the absolute path to the module root.
func FindModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached filesystem root
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// EnsureUnderBase cleans and anchors p under base, returning an absolute path.
// It errors if the resulting path escapes base.
func EnsureUnderBase(base, p string) (string, error) {
	cleaned := filepath.Clean(p)

	// If relative, anchor under base. If absolute, keep as-is.
	var candidate string
	if filepath.IsAbs(cleaned) {
		candidate = cleaned
	} else {
		candidate = filepath.Join(base, cleaned)
	}

	// Resolve symlinks ONLY on the candidate, not on the base.
	candResolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		// If the file itself doesn't exist yet, EvalSymlinks may fail.
		// Fall back to the cleaned candidate for comparison and let the open/read fail later.
		candResolved = candidate
	}

	// Compute absolutes (no filesystem access needed for base).
	baseAbs := filepath.Clean(base)
	if !filepath.IsAbs(baseAbs) {
		baseAbs, err = filepath.Abs(baseAbs)
		if err != nil {
			return "", err
		}
	}
	candAbs := filepath.Clean(candResolved)
	if !filepath.IsAbs(candAbs) {
		candAbs, err = filepath.Abs(candAbs)
		if err != nil {
			return "", err
		}
	}

	sep := string(os.PathSeparator)
	// Normalize base to have trailing separator for robust prefix checking.
	basePrefix := baseAbs
	if !strings.HasSuffix(basePrefix, sep) {
		basePrefix += sep
	}

	if candAbs == baseAbs || strings.HasPrefix(candAbs, basePrefix) {
		return candAbs, nil
	}
	return "", fmt.Errorf("invalid path: outside allowed base directory")
}
