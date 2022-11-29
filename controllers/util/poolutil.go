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
	"fmt"
	"github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"io"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"
)

const (
	PoolConfigmapName     = "ibm-flashsystem-pools"
	FSCConfigmapMountPath = "/config"

	TopologySecretDataKey        = "config"
	TopologyStorageClassByMgmtId = "by_management_id"

	PVMgmtAddrKey = "array_address"

	CsiIBMBlockDriver = "block.csi.ibm.com"
	CsiIBMBlockScPool = "pool"

	PersistentVolumeClaimKind = "PersistentVolumeClaim"

	DefaultSecretNameKey      = "csi.storage.k8s.io/secret-name"      // #nosec G101 - false positive
	DefaultSecretNamespaceKey = "csi.storage.k8s.io/secret-namespace" // #nosec G101 - false positive

	ProvisionerSecretNameKey      = "csi.storage.k8s.io/provisioner-secret-name"      // #nosec G101 - false positive
	ProvisionerSecretNamespaceKey = "csi.storage.k8s.io/provisioner-secret-namespace" // #nosec G101 - false positive

	SecretManagementAddressKey = "management_address" // #nosec G101 - false positive
)

type FSCMatchNotFoundError struct{}

func (m *FSCMatchNotFoundError) Error() string {
	return "no matching FlashSystemCluster found"
}

type UniqueFSCMatchError struct{}

func (m *UniqueFSCMatchError) Error() string {
	return "cannot find unique FlashSystemCluster"
}

type FlashSystemClusterMapContent struct {
	ScPoolMap map[string]string `json:"storageclass"`
	Secret    string            `json:"secret"`
}

type FSCConfigMapData struct {
	FlashSystemClusterMap map[string]FlashSystemClusterMapContent
}

var IgnoreUpdateAndGenericPredicate = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		return true
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		return true
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		return false
	},
	GenericFunc: func(e event.GenericEvent) bool {
		return false
	},
}

var SecretMgmtAddrPredicate = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		secret, ok := e.Object.(*corev1.Secret)
		if !ok {
			return false
		}
		_, exist := secret.Data[SecretManagementAddressKey]
		return exist
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		secret, ok := e.Object.(*corev1.Secret)
		if !ok {
			return false
		}
		_, exist := secret.Data[SecretManagementAddressKey]
		return exist
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldSecret, ok := e.ObjectOld.(*corev1.Secret)
		newSecret := e.ObjectNew.(*corev1.Secret)
		if !ok {
			return false
		}

		oldMgmtAddr, exist1 := oldSecret.Data[SecretManagementAddressKey]
		newMgmtAddr, exist2 := newSecret.Data[SecretManagementAddressKey]

		return exist1 && exist2 && (string(oldMgmtAddr) != string(newMgmtAddr))
	},
	GenericFunc: func(e event.GenericEvent) bool {
		return false
	},
}

var RunDeletePredicate = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		return false
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		return true
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		return false
	},
	GenericFunc: func(e event.GenericEvent) bool {
		return false
	},
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

func MapClustersByMgmtAddress(client client.Client, logger logr.Logger) (map[string]v1alpha1.FlashSystemCluster, error) {
	clusters := &v1alpha1.FlashSystemClusterList{}
	if err := client.List(context.Background(), clusters); err != nil {
		logger.Error(nil, "failed to list FlashSystemClusters")
		return nil, err
	}
	clustersMapByMgmtAddr := make(map[string]v1alpha1.FlashSystemCluster)
	for _, cluster := range clusters.Items {
		clusterSecret := &corev1.Secret{}
		err := client.Get(context.Background(),
			types.NamespacedName{
				Namespace: cluster.Spec.Secret.Namespace,
				Name:      cluster.Spec.Secret.Name},
			clusterSecret)
		if err != nil {
			logger.Error(nil, "failed to get FlashSystemCluster secret")
			return nil, err
		}
		clusterSecretManagementAddress := clusterSecret.Data[SecretManagementAddressKey]
		clustersMapByMgmtAddr[string(clusterSecretManagementAddress)] = cluster
	}
	return clustersMapByMgmtAddr, nil
}

func GetStorageClassSecretNamespacedName(sc *storagev1.StorageClass) (string, string, error) {
	secretName, secretNamespace := sc.Parameters[DefaultSecretNameKey], sc.Parameters[DefaultSecretNamespaceKey]
	if secretName == "" || secretNamespace == "" {
		secretName, secretNamespace = sc.Parameters[ProvisionerSecretNameKey], sc.Parameters[ProvisionerSecretNamespaceKey]
		if secretName == "" || secretNamespace == "" {
			return "", "", fmt.Errorf("failed to find secret name or namespace in StorageClass")
		}
	}
	return secretName, secretNamespace, nil
}

func GetStorageClassSecret(client client.Client, sc *storagev1.StorageClass) (corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretName, secretNamespace, err := GetStorageClassSecretNamespacedName(sc)
	if err != nil {
		return *secret, err
	}

	err = client.Get(context.Background(),
		types.NamespacedName{Namespace: secretNamespace, Name: secretName},
		secret)

	return *secret, err
}
