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

package storageclass

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

func reconcileSC(obj runtime.Object) bool {
	sc, ok := obj.(*storagev1.StorageClass)
	if !ok {
		return false
	}

	if sc.Provisioner == util.CsiIBMBlockDriver {
		return true
	}

	return false
}

var scPredicate = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		return reconcileSC(e.Object)
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		return reconcileSC(e.Object)
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		return reconcileSC(e.ObjectNew)
	},
	GenericFunc: func(e event.GenericEvent) bool {
		return reconcileSC(e.Object)
	},
}

type StorageClassWatcher struct {
	Client    client.Client
	Scheme    *runtime.Scheme
	Log       logr.Logger
	Namespace string
}

func (r *StorageClassWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1.StorageClass{}, builder.WithPredicates(scPredicate)).
		Complete(r)
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch

// Reconcile StorageClass
func (r *StorageClassWatcher) Reconcile(_ context.Context, request reconcile.Request) (result reconcile.Result, err error) {
	result = reconcile.Result{}
	prevLogger := r.Log

	defer func() {
		r.Log = prevLogger
		if errors.IsConflict(err) {
			r.Log.Info("requeue due to resource update conflicts")
			result = reconcile.Result{Requeue: true}
			err = nil
		}
	}()

	r.Log = r.Log.WithValues("StorageClass", request.NamespacedName)
	r.Log.Info("reconciling StorageClass")

	configMap, err := util.GetCreateConfigmap(r.Client, r.Log, r.Namespace, false)
	if err != nil {
		return result, err
	}

	sc := &storagev1.StorageClass{}
	err = r.Client.Get(context.TODO(), request.NamespacedName, sc)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("StorageClass not found")
			for fscName := range configMap.Data {
				err = r.removeStorageClassFromConfigMap(*configMap, fscName, request.Name)
				if err != nil {
					return result, err
				}
			}
			return result, nil
		}
		return result, err
	}

	flashSystemCluster, fscErr := r.getFlashSystemClusterByStorageClass(sc)
	if fscErr == nil {
		fscName := flashSystemCluster.GetName()
		r.Log.Info("FlashSystemCluster found", "FlashSystemCluster", fscName)

		// Check GetDeletionTimestamp to determine if the object is under deletion
		if !sc.GetDeletionTimestamp().IsZero() {
			r.Log.Info("object is terminated")
			err = r.removeStorageClassFromConfigMap(*configMap, fscName, request.Name)
			if err != nil {
				return result, err
			}
		} else {
			poolName, ok := sc.Parameters[util.CsiIBMBlockScPool]
			if ok {
				err = r.addStorageClassToConfigMap(*configMap, fscName, request.Name, poolName)
				if err != nil {
					return result, err
				}
			} else {
				r.Log.Error(nil, "cannot reconcile StorageClass without a pool")
			}
		}
		return result, nil
	} else {
		r.Log.Info("cannot find FlashSystemCluster for StorageClass")
		return result, fscErr
	}
}

func (r *StorageClassWatcher) getFlashSystemClusterByStorageClass(sc *storagev1.StorageClass) (v1alpha1.FlashSystemCluster, error) {
	r.Log.Info("looking for FlashSystemCluster by StorageClass")
	storageClassSecret := &corev1.Secret{}
	foundCluster := v1alpha1.FlashSystemCluster{}
	err := r.Client.Get(context.Background(),
		types.NamespacedName{
			Namespace: sc.Parameters[util.SecretNamespaceKey],
			Name:      sc.Parameters[util.SecretNameKey]},
		storageClassSecret)
	if err != nil {
		r.Log.Error(nil, "failed to find StorageClass secret")
		return foundCluster, err
	}
	secretManagementAddress := storageClassSecret.Data[util.SecretManagementAddressKey]

	clusters := &v1alpha1.FlashSystemClusterList{}
	err = r.Client.List(context.Background(), clusters)
	if err != nil {
		r.Log.Error(nil, "failed to list FlashSystemClusterList")
		return foundCluster, err
	}

	for _, c := range clusters.Items {
		clusterSecret := &corev1.Secret{}
		err = r.Client.Get(context.Background(),
			types.NamespacedName{
				Namespace: c.Spec.Secret.Namespace,
				Name:      c.Spec.Secret.Name},
			clusterSecret)
		if err != nil {
			r.Log.Error(nil, "failed to get FlashSystemCluster secret")
			return foundCluster, err
		}
		clusterSecretManagement := clusterSecret.Data[util.SecretManagementAddressKey]
		secretsEqual := bytes.Compare(clusterSecretManagement, secretManagementAddress)
		if secretsEqual == 0 {
			r.Log.Info("found StorageClass with a matching secret address")
			return c, nil
		}
	}
	msg := "failed to match StorageClass to FlashSystemCluster item"
	r.Log.Error(nil, msg)
	return foundCluster, fmt.Errorf(msg)
}

func (r *StorageClassWatcher) getManagementMapFromSecret(topologySecret corev1.Secret) (map[string]v1alpha1.FlashSystemCluster, error) {
	clustersByMgmtAddr := make(map[string]v1alpha1.FlashSystemCluster)
	jsonSecretConfigData := make(map[string]map[string]string)
	topologySecretData := topologySecret.Data["config"]

	// decoding topologySecretData from base64 to json
	decodedSecretData, err := base64.StdEncoding.DecodeString(string(topologySecretData))
	if err != nil {
		r.Log.Error(nil, "failed to decode configData from topology secret")
		return clustersByMgmtAddr, err
	}

	if err := json.Unmarshal(decodedSecretData, &jsonSecretConfigData); err != nil {
		r.Log.Error(nil, "failed to unmarshal the decoded configData from topology secret")
		return clustersByMgmtAddr, err
	}
	clusters := &v1alpha1.FlashSystemClusterList{}
	if err := r.Client.List(context.Background(), clusters); err != nil {
		r.Log.Error(nil, "failed to list FlashSystemClusterList for topology secret")
	}
	clusterMap := make(map[string]v1alpha1.FlashSystemCluster)
	for _, cluster := range clusters.Items {
		clusterSecret := &corev1.Secret{}
		if err := r.Client.Get(context.Background(),
			types.NamespacedName{
				Namespace: cluster.Spec.Secret.Namespace,
				Name:      cluster.Spec.Secret.Name},
			clusterSecret); err != nil {
			r.Log.Error(nil, "failed to get FlashSystemCluster secret for topology secret")
			return clustersByMgmtAddr, err
		}
		clusterSecretManagementAddress := clusterSecret.Data[util.SecretManagementAddressKey]
		clusterMap[string(clusterSecretManagementAddress)] = cluster
	}
	for _, managementData := range jsonSecretConfigData {
		managementAddress := managementData["management_address"]
		// checking if managementAddress is in clusterMap and if it is, adding it to the final managementMap
		if cluster, ok := clusterMap[managementAddress]; ok {
			r.Log.Info("found FlashSystemCluster with a matching secret address for topology secret")
			clustersByMgmtAddr[managementAddress] = cluster
		}
	}
	return clustersByMgmtAddr, nil
}

func (r *StorageClassWatcher) removeStorageClassFromConfigMap(configMap corev1.ConfigMap, fscName string, scName string) error {
	r.Log.Info("removing StorageClass from pools ConfigMap", "FlashSystemCluster", fscName, "ConfigMap", configMap.Name)

	fscContent, exist := configMap.Data[fscName]
	if exist {
		var fsMap util.FlashSystemClusterMapContent
		err := json.Unmarshal([]byte(fscContent), &fsMap)
		if err != nil {
			r.Log.Error(err, "failed to unmarshal value from pools ConfigMap", "ConfigMap", configMap.Name)
			return err
		}

		delete(fsMap.ScPoolMap, scName)
		val, err := json.Marshal(fsMap)
		if err != nil {
			return err
		}
		configMap.Data[fscName] = string(val)
		err = r.Client.Update(context.TODO(), &configMap)
		if err != nil {
			r.Log.Error(err, "failed to update pools ConfigMap", "ConfigMap", configMap.Name)
			return err
		}
	}
	return nil
}

func (r *StorageClassWatcher) addStorageClassToConfigMap(configMap corev1.ConfigMap, fscName string, scName string, poolName string) error {
	r.Log.Info("adding StorageClass to pools ConfigMap", "FlashSystemCluster", fscName, "ConfigMap", configMap.Name)

	fscContent, exist := configMap.Data[fscName]
	if !exist {
		msg := "failed to get FlashSystemCluster entry from pools ConfigMap"
		r.Log.Error(nil, msg, "FlashSystemCluster", fscName, "ConfigMap", configMap.Name)
		return fmt.Errorf(msg)
	}

	var fsMap util.FlashSystemClusterMapContent
	err := json.Unmarshal([]byte(fscContent), &fsMap)
	if err != nil {
		r.Log.Error(err, "failed to unmarshal value from pools ConfigMap", "ConfigMap", configMap.Name)
		return err
	}

	fsMap.ScPoolMap[scName] = poolName
	val, err := json.Marshal(fsMap)
	if err != nil {
		return err
	}
	configMap.Data[fscName] = string(val)

	err = r.Client.Update(context.TODO(), &configMap)
	if err != nil {
		r.Log.Error(err, "failed to update pools ConfigMap", "ConfigMap", configMap.Name)
		return err
	}

	return nil
}
