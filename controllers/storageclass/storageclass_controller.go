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
	apiv1 "github.com/IBM/ibm-storage-odf-operator/api/v1"
	"github.com/IBM/ibm-storage-odf-operator/controllers"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
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
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

type storageClassMapper struct {
	reconciler *StorageClassWatcher
}

func (f *storageClassMapper) ConfigMapToStorageClassMapFunc(object client.Object) []reconcile.Request {
	requests := []reconcile.Request{}
	if object.GetName() == util.PoolConfigmapName {
		f.reconciler.Log.Info("Discovered ODF-FS configMap deletion. Reconciling all storageClasses", "ConfigMapToStorageClassMapFunc", f)

		storageClasses := &storagev1.StorageClassList{}
		err := f.reconciler.Client.List(context.TODO(), storageClasses)
		if err != nil {
			f.reconciler.Log.Error(err, "failed to list StorageClasses", "ConfigMapToStorageClassMapFunc", f)
			return nil
		}

		for _, sc := range storageClasses.Items {
			if sc.Provisioner == util.CsiIBMBlockDriver {
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: sc.GetNamespace(),
						Name:      sc.GetName(),
					},
				}
				requests = append(requests, req)
			}
		}
	}
	return requests
}

func (f *storageClassMapper) fscStorageClassMap(_ client.Object) []reconcile.Request {
	storageClasses := &storagev1.StorageClassList{}
	err := f.reconciler.Client.List(context.TODO(), storageClasses)
	if err != nil {
		f.reconciler.Log.Error(err, "failed to list StorageClasses", "storageClassMapper", f)
		return nil
	}

	requests := []reconcile.Request{}
	for _, sc := range storageClasses.Items {
		if sc.Provisioner == util.CsiIBMBlockDriver {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: sc.GetNamespace(),
					Name:      sc.GetName(),
				},
			}
			requests = append(requests, req)
		}
	}
	return requests
}

func (f *storageClassMapper) secretStorageClassMap(object client.Object) []reconcile.Request {
	requests := []reconcile.Request{}
	secret, ok := object.(*corev1.Secret)
	if !ok {
		return requests
	}

	storageClasses := &storagev1.StorageClassList{}
	err := f.reconciler.Client.List(context.TODO(), storageClasses)
	if err != nil {
		f.reconciler.Log.Error(err, "failed to list StorageClasses", "storageClassMapper", f)
		return nil
	}

	isFscSecret := controllers.IsFSCSecret(secret.Name)
	for i := range storageClasses.Items {
		sc := storageClasses.Items[i]
		if sc.Provisioner == util.CsiIBMBlockDriver {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: sc.GetNamespace(),
					Name:      sc.GetName(),
				},
			}

			if isFscSecret {
				requests = append(requests, req)
			} else {
				scSecretName, scSecretNameSpace, err := util.GetStorageClassSecretNamespacedName(&sc)
				if err == nil {
					if scSecretName == secret.Name && scSecretNameSpace == secret.Namespace {
						requests = append(requests, req)
					}
				}
			}
		}
	}
	return requests
}

func (r *StorageClassWatcher) SetupWithManager(mgr ctrl.Manager) error {
	scMapper := &storageClassMapper{
		reconciler: r,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1.StorageClass{}, builder.WithPredicates(scPredicate)).
		Watches(&source.Kind{
			Type: &apiv1.FlashSystemCluster{},
		}, handler.EnqueueRequestsFromMapFunc(scMapper.fscStorageClassMap), builder.WithPredicates(util.IgnoreUpdateAndGenericPredicate)).
		Watches(&source.Kind{
			Type: &corev1.ConfigMap{},
		}, handler.EnqueueRequestsFromMapFunc(scMapper.ConfigMapToStorageClassMapFunc), builder.WithPredicates(util.RunDeletePredicate)).
		Watches(&source.Kind{
			Type: &corev1.Secret{},
		}, handler.EnqueueRequestsFromMapFunc(scMapper.secretStorageClassMap), builder.WithPredicates(util.SecretMgmtAddrPredicate)).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
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
	if err = r.Client.Get(context.TODO(), request.NamespacedName, sc); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("StorageClass not found")
			for fscName := range configMap.Data {
				if err = r.removeStorageClassFromConfigMap(*configMap, fscName, request.Name); err != nil {
					return result, err
				}
			}
			return result, nil
		}
		return result, err
	}

	if err = r.ensureConfigMapUpdated(request, sc, configMap); err != nil {
		return result, err
	}

	return result, nil
}

func (r *StorageClassWatcher) ensureConfigMapUpdated(request reconcile.Request, sc *storagev1.StorageClass, configMap *corev1.ConfigMap) error {
	_, isTopology := sc.Parameters[util.TopologyStorageClassByMgmtId]
	fscToPoolsMap, fscErr := r.getFlashSystemClusterByStorageClass(sc, isTopology)

	if fscErr != nil {
		r.Log.Error(nil, "error while trying to match StorageClass to FlashSystemCluster")
		return fscErr
	}

	for fscName := range fscToPoolsMap {
		_, fscExist := configMap.Data[fscName]
		if !fscExist {
			msg := "failed to get FlashSystemCluster entry from pools ConfigMap"
			r.Log.Error(nil, msg, "FlashSystemCluster", fscName, "ConfigMap", configMap.Name)
			return fmt.Errorf("%s", msg)
		}
	}

	isSCDeleted := !sc.GetDeletionTimestamp().IsZero()
	for fscName := range configMap.Data {
		if isSCDeleted {
			r.Log.Info("object is terminated")
			if err := r.removeStorageClassFromConfigMap(*configMap, fscName, request.Name); err != nil {
				return err
			}
		} else {
			poolName, fscExist := fscToPoolsMap[fscName]
			if fscExist {
				if poolName == "" {
					r.Log.Error(nil, "cannot reconcile StorageClass without a pool")
					return nil
				}
				if err := r.addStorageClassToConfigMap(*configMap, fscName, request.Name, poolName); err != nil {
					return err
				}
			} else {
				if err := r.removeStorageClassFromConfigMap(*configMap, fscName, request.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *StorageClassWatcher) getFlashSystemClusterByStorageClass(sc *storagev1.StorageClass, isTopology bool) (map[string]string, error) {
	fscToPoolsMap := make(map[string]string)
	r.Log.Info("looking for FlashSystemCluster by StorageClass")
	storageClassSecret, err := util.GetStorageClassSecret(r.Client, sc)
	if err != nil {
		r.Log.Error(nil, "failed to find StorageClass secret")
		return fscToPoolsMap, err
	}

	if isTopology {
		fscToPoolsMap, err = r.getFscByTopologyStorageClass(sc, storageClassSecret)
	} else {
		fscToPoolsMap, err = r.getFscByRegularStorageClass(sc, storageClassSecret)
	}

	if err != nil {
		return fscToPoolsMap, err
	}
	return fscToPoolsMap, nil
}

func (r *StorageClassWatcher) getFscByTopologyStorageClass(sc *storagev1.StorageClass, storageClassSecret corev1.Secret) (map[string]string, error) {
	fscToPoolsMap := make(map[string]string)
	r.Log.Info("StorageClass is a topology aware StorageClass")
	clustersMapByMgmtId, err := r.mapClustersByMgmtId(&storageClassSecret)
	if err != nil {
		r.Log.Error(nil, "failed to get management map from topology secret from StorageClass")
		return fscToPoolsMap, err
	}

	byMgmtIdDataOfSc := sc.Parameters[util.TopologyStorageClassByMgmtId]
	var mgmtDataByMgmtId map[string]interface{}
	if err := json.Unmarshal([]byte(byMgmtIdDataOfSc), &mgmtDataByMgmtId); err != nil {
		r.Log.Error(nil, "failed to unmarshal the topology storage class \"by_management_id\" parameter data")
		return fscToPoolsMap, err
	}

	for mgmtId, fsc := range clustersMapByMgmtId {
		poolName, err := r.extractPoolName(*sc, mgmtDataByMgmtId, mgmtId)
		if err != nil {
			r.Log.Error(nil, "failed to extract pool name from topology storage class")
			return fscToPoolsMap, err
		}
		fscToPoolsMap[fsc.Name] = poolName
	}

	return fscToPoolsMap, nil
}

func (r *StorageClassWatcher) getFscByRegularStorageClass(sc *storagev1.StorageClass, storageClassSecret corev1.Secret) (map[string]string, error) {
	fscToPoolsMap := make(map[string]string)
	clusters := &apiv1.FlashSystemClusterList{}
	if err := r.Client.List(context.Background(), clusters); err != nil {
		r.Log.Error(nil, "failed to list FlashSystemClusterList")
		return fscToPoolsMap, err
	}

	secretManagementAddress := storageClassSecret.Data[util.SecretManagementAddressKey]
	for _, c := range clusters.Items {
		clusterSecret := &corev1.Secret{}
		err := r.Client.Get(context.Background(),
			types.NamespacedName{
				Namespace: c.Spec.Secret.Namespace,
				Name:      c.Spec.Secret.Name},
			clusterSecret)
		if err != nil {
			r.Log.Error(nil, "failed to get FlashSystemCluster secret")
			return fscToPoolsMap, err
		}
		clusterSecretManagementAddress := clusterSecret.Data[util.SecretManagementAddressKey]
		secretsEqual := bytes.Compare(clusterSecretManagementAddress, secretManagementAddress)
		if secretsEqual == 0 {
			r.Log.Info("found StorageClass with a matching secret address")
			poolName := sc.Parameters[util.CsiIBMBlockScPool]
			fscToPoolsMap[c.Name] = poolName
			return fscToPoolsMap, nil
		}
	}

	return fscToPoolsMap, nil
}

func (r *StorageClassWatcher) extractPoolName(sc storagev1.StorageClass, mgmtDataByMgmtId map[string]interface{}, mgmtId string) (string, error) {
	r.Log.Info("extracting the pool name from StorageClass with management id: ", "management id", mgmtId)
	poolName := ""
	mgmtData, ok := mgmtDataByMgmtId[mgmtId]
	if !ok {
		msg := "failed to find the management id in the \"by_management_id\" parameter in the StorageClass"
		r.Log.Error(nil, msg, "management id", mgmtId)
		return poolName, fmt.Errorf("%s", msg)
	}
	byMgmtIdData := mgmtData.(map[string]interface{})
	poolName, ok = byMgmtIdData[util.CsiIBMBlockScPool].(string)
	if !ok {
		poolName = sc.Parameters[util.CsiIBMBlockScPool]
	}
	return poolName, nil
}

func (r *StorageClassWatcher) mapClustersByMgmtId(topologySecret *corev1.Secret) (map[string]apiv1.FlashSystemCluster, error) {
	clustersByMgmtId := make(map[string]apiv1.FlashSystemCluster)
	secretMgmtDataByMgmtId := make(map[string]interface{})
	topologySecretData := string(topologySecret.Data[util.TopologySecretDataKey])

	if err := json.Unmarshal([]byte(topologySecretData), &secretMgmtDataByMgmtId); err != nil {
		r.Log.Error(nil, "failed to unmarshal the decoded configData from topology secret")
		return clustersByMgmtId, err
	}
	clustersMapByMgmtAddr, err := util.MapClustersByMgmtAddress(r.Client, r.Log)
	if err != nil {
		r.Log.Error(nil, "failed to get clusters mapped by mgmt address")
		return clustersByMgmtId, err
	}
	for mgmtId, mgmtData := range secretMgmtDataByMgmtId {
		mgmtAddress := mgmtData.(map[string]interface{})[util.SecretManagementAddressKey].(string)
		if cluster, ok := clustersMapByMgmtAddr[mgmtAddress]; ok {
			r.Log.Info("found FlashSystemCluster with a matching secret address for topology secret")
			clustersByMgmtId[mgmtId] = cluster
		}
	}
	return clustersByMgmtId, nil
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
		if err = r.Client.Update(context.TODO(), &configMap); err != nil {
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
		return fmt.Errorf("%s", msg)
	}

	var fsMap util.FlashSystemClusterMapContent
	if err := json.Unmarshal([]byte(fscContent), &fsMap); err != nil {
		r.Log.Error(err, "failed to unmarshal value from pools ConfigMap", "ConfigMap", configMap.Name)
		return err
	}

	fsMap.ScPoolMap[scName] = poolName
	val, err := json.Marshal(fsMap)
	if err != nil {
		return err
	}
	configMap.Data[fscName] = string(val)

	if err = r.Client.Update(context.TODO(), &configMap); err != nil {
		r.Log.Error(err, "failed to update pools ConfigMap", "ConfigMap", configMap.Name)
		return err
	}

	return nil
}
