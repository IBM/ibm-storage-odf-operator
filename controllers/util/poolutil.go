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
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	PoolConfigmapName     = "ibm-flashsystem-pools"
	FSCConfigmapMountPath = "/config"
	TopologySecretDataKey = "config"

	CsiIBMBlockDriver = "block.csi.ibm.com"
	CsiIBMBlockScPool = "pool"

	SecretNameKey              = "csi.storage.k8s.io/provisioner-secret-name"      // #nosec G101 - false positive
	SecretNamespaceKey         = "csi.storage.k8s.io/provisioner-secret-namespace" // #nosec G101 - false positive
	SecretManagementAddressKey = "management_address"                              // #nosec G101 - false positive
)

type FlashSystemClusterMapContent struct {
	ScPoolMap map[string]string `json:"storageclass"`
	Secret    string            `json:"secret"`
}

type FSCConfigMapData struct {
	FlashSystemClusterMap map[string]FlashSystemClusterMapContent
}

func ReadPoolConfigMapFile() (map[string]FlashSystemClusterMapContent, error) {
	var flashSystemClustersMap = make(map[string]FlashSystemClusterMapContent)
	var flashSystemClusterContent FlashSystemClusterMapContent
	fscPath := FSCConfigmapMountPath + "/"

	files, err := os.ReadDir(fscPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			flashSystemClusterContent, err = getFileContent(filepath.Join(fscPath, file.Name()))
			if err != nil {
				return flashSystemClustersMap, err
			} else {
				flashSystemClustersMap[file.Name()] = flashSystemClusterContent
			}
		}
	}
	return flashSystemClustersMap, nil
}

func getFileContent(filePath string) (FlashSystemClusterMapContent, error) {
	var fscContent FlashSystemClusterMapContent
	fileReader, err := os.Open(filePath)
	if err != nil {
		return fscContent, err
	}

	fileContent, _ := io.ReadAll(fileReader)
	err = json.Unmarshal(fileContent, &fscContent)
	return fscContent, err
}

func GetCreateConfigmap(client client.Client, log logr.Logger, ns string, createIfMissing bool) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}

	err := client.Get(
		context.Background(),
		types.NamespacedName{Namespace: ns, Name: PoolConfigmapName},
		configMap)

	if err != nil {
		if errors.IsNotFound(err) && createIfMissing {
			configMap = initScPoolConfigMap(ns)
			configMap.Data = make(map[string]string)
			log.Info("creating pools ConfigMap", "ConfigMap", PoolConfigmapName)
			err = client.Create(context.Background(), configMap)
			if err != nil {
				log.Error(err, "failed to create pools ConfigMap", "ConfigMap", PoolConfigmapName)
				return nil, err
			}
		} else {
			log.Error(err, "failed to get pools ConfigMap", "ConfigMap", PoolConfigmapName)
			return nil, err
		}
	}
	return configMap, err
}

func initScPoolConfigMap(ns string) *corev1.ConfigMap {
	selectLabels := GetLabels()
	scPoolConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PoolConfigmapName,
			Namespace: ns,
			Labels:    selectLabels,
		},
	}
	return scPoolConfigMap
}
