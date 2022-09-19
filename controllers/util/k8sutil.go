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

package util

import (
	"fmt"
	"os"
)

// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
// which is the namespace where the watch activity happens.
// this value is empty if the operator is running with clusterScope.
const WatchNamespaceEnvVar = "WATCH_NAMESPACE"
const ExporterImageEnvVar = "EXPORTER_IMAGE"

type Label struct {
	Name  string
	Value string
}

var OdfLabel = Label{"odf", "storage.ibm.com"}
var ComponentLabel = Label{"app.kubernetes.io/component", "ibm-storage-odf-operator"}
var OdfFsLabel = Label{"odf-fs", "odf-fs-workspace"}

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}
	return ns, nil
}

// GetExporterImage returns the exporter image from operator env by OLM bundle
func GetExporterImage() (string, error) {
	image, found := os.LookupEnv(ExporterImageEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", ExporterImageEnvVar)
	}
	return image, nil
}

// GetLabels returns the labels with cluster name
func GetLabels() map[string]string {
	return map[string]string{
		ComponentLabel.Name: ComponentLabel.Value,
		OdfFsLabel.Name:     OdfFsLabel.Value,
	}
}

//// CalculateDataHash generates a sha256 hex-digest for a data object
//func CalculateDataHash(dataObject interface{}) (string, error) {
//	data, err := json.Marshal(dataObject)
//	if err != nil {
//		return "", err
//	}
//
//	hash := sha256.New()
//	hash.Write(data)
//	return hex.EncodeToString(hash.Sum(nil)), nil
//}

func IsContain(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func Remove(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
