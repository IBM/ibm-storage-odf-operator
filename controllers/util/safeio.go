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
    "errors"
    "os"
    "path/filepath"
    "strings"
)

// ensureUnderBase cleans and anchors p under base, returning an absolute path.
// It errors if the resulting path escapes base.
func ensureUnderBase(base, p string) (string, error) {
    // Normalize the candidate path.
    cleaned := filepath.Clean(p)

    // If relative, anchor under base. If absolute, keep it but still validate.
    var joined string
    if filepath.IsAbs(cleaned) {
        joined = cleaned
    } else {
        joined = filepath.Join(base, cleaned)
    }

    // Resolve absolute paths for robust prefix comparison.
    baseAbs, err := filepath.Abs(base)
    if err != nil {
        return "", err
    }
    // Optional: resolve symlinks inside the candidate, then Clean again.
    // This helps with cases where a symlink could point outside `base`.
    // Gosec recognizes EvalSymlinks as part of the sanitization process.
    // See: securego G304 guidance.
    // joined, err = filepath.EvalSymlinks(joined)
    // if err != nil {
    //     return "", err
    // }

    joinedAbs, err := filepath.Abs(filepath.Clean(joined))
    if err != nil {
        return "", err
    }

    sep := string(os.PathSeparator)
    if !strings.HasPrefix(joinedAbs, baseAbs+sep) && joinedAbs != baseAbs {
        return "", errors.New("invalid path: outside allowed base directory")
    }

    return joinedAbs, nil
}
