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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	util.FSCConfigMapData
}

func (r *StorageClassWatcher) SetupWithManager(mgr ctrl.Manager) error {
	r.FlashSystemClusterMap = make(map[string]util.FlashSystemClusterMapContent)
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

	r.Log = r.Log.WithValues("Request.Name", request.NamespacedName)

	sc := &storagev1.StorageClass{}
	err = r.Client.Get(context.TODO(), request.NamespacedName, sc)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("StorageClass not found", "sc", request.Name)
			for _, fscContent := range r.FlashSystemClusterMap {
				delete(fscContent.ScPoolMap, request.Name)
			}
			err = r.updateConfigmap()
			if err != nil {
				return result, err
			} else {
				return result, nil
			}
		}
		return result, err
	}

	if flashSystemCluster, fscErr := r.getFlashSystemClusterByStorageClass(sc); fscErr == nil {
		fscName := flashSystemCluster.GetName()
		fscSecretName := flashSystemCluster.Spec.Secret.Name
		r.Log.Info("FlashsystemCluster found", "flashsystemcluster", fscName)

		// Check GetDeletionTimestamp to determine if the object is under deletion
		if !sc.GetDeletionTimestamp().IsZero() {
			r.Log.Info("Object is terminated")
			delete(r.FlashSystemClusterMap[fscName].ScPoolMap, request.Name)

			err = r.updateConfigmap()
			if err != nil {
				return result, err
			} else {
				return result, nil
			}
		}

		_, ok := r.FlashSystemClusterMap[fscName].ScPoolMap[request.Name]
		if ok {
			r.Log.Info("Reconciling a existing StorageClass: ", "sc", request.Name)
			delete(r.FlashSystemClusterMap[fscName].ScPoolMap, request.Name)
		}

		poolName, ok := sc.Parameters[util.CsiIBMBlockScPool]
		if ok {
			if _, exist := r.FlashSystemClusterMap[fscName]; !exist {
				r.FlashSystemClusterMap[fscName] = util.FlashSystemClusterMapContent{
					ScPoolMap: make(map[string]string), Secret: fscSecretName}
			}
			r.FlashSystemClusterMap[fscName].ScPoolMap[request.Name] = poolName

		} else {
			r.Log.Error(nil, "Reconciling a StorageClass without a pool", "sc", request.Name)
		}

		err = r.updateConfigmap()
		if err != nil {
			r.Log.Error(err, "Failed to update configmap")
			return result, err
		}

		r.Log.Info("Reconciling StorageClass")
	} else {
		r.Log.Info("Cannot find FlashsystemCluster for StorageClass", "sc", request.Name)
	}
	return result, nil
}

func (r *StorageClassWatcher) getFlashSystemClusterByStorageClass(sc *storagev1.StorageClass) (v1alpha1.FlashSystemCluster, error) {
	r.Log.Info("Looking for flashSystemCluster by storageClass", "sc", sc.Name)
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

func InitScPoolConfigMap(ns string) *corev1.ConfigMap {
	selectLabels := util.GetLabels("")

	scPoolConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.PoolConfigmapName,
			Namespace: ns,
			Labels:    selectLabels,
		},
	}
	return scPoolConfigMap
}

func (r *StorageClassWatcher) getCreateConfigmap() (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}

	err := r.Client.Get(
		context.Background(),
		types.NamespacedName{Namespace: r.Namespace, Name: util.PoolConfigmapName},
		configMap)

	if err != nil {
		if errors.IsNotFound(err) {
			configMap = InitScPoolConfigMap(r.Namespace)
			err = r.Client.Create(context.Background(), configMap)
			if err != nil {
				r.Log.Error(err, "create configMap failed", "cm", util.PoolConfigmapName)
				return nil, err
			}
		} else {
			r.Log.Error(err, "get configMap failed", "cm", util.PoolConfigmapName)
			return nil, err
		}
	}

	return configMap, nil
}

func (r *StorageClassWatcher) updateConfigmap() error {
	configMap, err := r.getCreateConfigmap()
	if err != nil {
		return err
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	value, err := util.GenerateFSCConfigmapContent(r.FSCConfigMapData)
	if err != nil {
		r.Log.Error(err, "configMap marshal failed")
		return err
	} else {
		configMap.Data = value
	}

	err = r.Client.Update(context.Background(), configMap)
	if err != nil {
		r.Log.Error(err, "configMap update failed")
		return err
	}

	return nil
}
