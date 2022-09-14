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

// reconcile StorageClass
func (r *StorageClassWatcher) Reconcile(_ context.Context, request reconcile.Request) (result reconcile.Result, err error) {
	result = reconcile.Result{}
	prevLogger := r.Log

	defer func() {
		r.Log = prevLogger
		if errors.IsConflict(err) {
			r.Log.Info("Requeue due to resource update conflicts")
			result = reconcile.Result{Requeue: true}
			err = nil
		}
	}()

	r.Log = r.Log.WithValues("StorageClass", request.NamespacedName)
	r.Log.Info("Reconciling StorageClass")

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
			r.Log.Info("Object is terminated")
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
				r.Log.Error(nil, "Cannot reconcile StorageClass without a pool")
			}
		}
		return result, nil
	} else {
		r.Log.Info("Cannot find FlashSystemCluster for StorageClass")
		return result, fscErr
	}
}

func (r *StorageClassWatcher) getFlashSystemClusterByStorageClass(sc *storagev1.StorageClass) (v1alpha1.FlashSystemCluster, error) {
	r.Log.Info("Looking for FlashSystemCluster by StorageClass", "sc", sc.Name)
	storageClassSecret := &corev1.Secret{}
	foundCluster := v1alpha1.FlashSystemCluster{}
	err := r.Client.Get(context.Background(),
		types.NamespacedName{
			Namespace: sc.Parameters[util.SecretNamespaceKey],
			Name:      sc.Parameters[util.SecretNameKey]},
		storageClassSecret)
	if err != nil {
		r.Log.Error(nil, "failed to find storageClass secret", "sc", sc.Name)
		return foundCluster, err
	}
	secretManagementAddress := storageClassSecret.Data[util.SecretManagementAddressKey]

	clusters := &v1alpha1.FlashSystemClusterList{}
	err = r.Client.List(context.Background(), clusters)
	if err != nil {
		r.Log.Error(nil, "failed to list FlashSystemClusterList", "sc", sc.Name)
		return foundCluster, err
	}

	for _, c := range clusters.Items {
		r.Log.Info("checking cluster", "cluster", c.Name)
		clusterSecret := &corev1.Secret{}
		err = r.Client.Get(context.Background(),
			types.NamespacedName{
				Namespace: c.Spec.Secret.Namespace,
				Name:      c.Spec.Secret.Name},
			clusterSecret)
		if err != nil {
			r.Log.Error(nil, "failed to FlashSystemCluster secret", "sc", c.Name)
			return foundCluster, err
		}
		clusterSecretManagement := clusterSecret.Data[util.SecretManagementAddressKey]
		secretsEqual := bytes.Compare(clusterSecretManagement, secretManagementAddress)
		if secretsEqual == 0 {
			r.Log.Info("found storageClass with a matching secret address", "sc", c.Name)
			return c, nil
		}
	}
	r.Log.Error(nil, "failed to match storageClass to flashSystemCluster item", "sc", sc.Name)
	return foundCluster, fmt.Errorf("failed to match storageClass to flashSystemCluster item")
}

func (r *StorageClassWatcher) removeStorageClassFromConfigMap(configMap corev1.ConfigMap, fscName string, scName string) error {
	r.Log.Info("Removing StorageClass from pools ConfigMap", "FlashSystemCluster", fscName, "ConfigMap", configMap.Name)

	fscContent, exist := configMap.Data[fscName]
	if exist {
		var fsMap util.FlashSystemClusterMapContent
		err := json.Unmarshal([]byte(fscContent), &fsMap)
		if err != nil {
			r.Log.Error(err, "Failed to unmarshal value from pools ConfigMap", "ConfigMap", configMap.Name)
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
			r.Log.Error(err, "Failed to update pools ConfigMap", "ConfigMap", configMap.Name)
			return err
		}
	}
	return nil
}

func (r *StorageClassWatcher) addStorageClassToConfigMap(configMap corev1.ConfigMap, fscName string, scName string, poolName string) error {
	r.Log.Info("Adding StorageClass to pools ConfigMap", "FlashSystemCluster", fscName, "ConfigMap", configMap.Name)

	fscContent, exist := configMap.Data[fscName]
	if !exist {
		r.Log.Error(nil, "Failed to get FlashSystemCluster entry from pools ConfigMap", "FlashSystemCluster", fscName, "ConfigMap", configMap.Name)
		return fmt.Errorf("failed to get FlashSystemCluster entry from configMap")
	}

	var fsMap util.FlashSystemClusterMapContent
	err := json.Unmarshal([]byte(fscContent), &fsMap)
	if err != nil {
		r.Log.Error(err, "Failed to unmarshal value from pools ConfigMap", "ConfigMap", configMap.Name)
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
		r.Log.Error(err, "Failed to update pools ConfigMap", "ConfigMap", configMap.Name)
		return err
	}

	return nil
}
