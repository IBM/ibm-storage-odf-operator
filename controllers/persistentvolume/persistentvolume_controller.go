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

package persistentvolume

import (
	"context"
	"github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

type persistentVolumeMapper struct {
	reconciler *PersistentVolumeWatcher
}

func (f *persistentVolumeMapper) pvMap(_ client.Object) []reconcile.Request {
	pvs := &corev1.PersistentVolumeList{}
	err := f.reconciler.Client.List(context.TODO(), pvs)
	if err != nil {
		f.reconciler.Log.Error(err, "failed to list PersistentVolumes", "pvMapper", f)
		return nil
	}

	requests := []reconcile.Request{}
	for _, pv := range pvs.Items {
		if isCSIBlockDriverPV(pv) {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pv.GetNamespace(),
					Name:      pv.GetName(),
				},
			}
			requests = append(requests, req)
		}
	}
	return requests
}

func isCSIBlockDriverPV(pv corev1.PersistentVolume) bool {
	return pv.Spec.CSI != nil && pv.Spec.CSI.Driver == util.CsiIBMBlockDriver && pv.Spec.ClaimRef != nil &&
		pv.Spec.ClaimRef.Kind == util.PersistentVolumeClaimKind
}

type PersistentVolumeWatcher struct {
	Client    client.Client
	Scheme    *runtime.Scheme
	Log       logr.Logger
	Namespace string
}

func (r *PersistentVolumeWatcher) SetupWithManager(mgr ctrl.Manager) error {
	pvMapper := &persistentVolumeMapper{
		reconciler: r,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.PersistentVolume{}).
		Watches(&source.Kind{
			Type: &v1alpha1.FlashSystemCluster{},
		}, handler.EnqueueRequestsFromMapFunc(pvMapper.pvMap), builder.WithPredicates(util.IgnoreUpdateAndGenericPredicate)).
		Watches(&source.Kind{
			Type: &corev1.Secret{},
		}, handler.EnqueueRequestsFromMapFunc(pvMapper.pvMap), builder.WithPredicates(util.IgnoreGenericPredicate)).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=odf.ibm.com,resources=flashsystemclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch;create;update;patch;delete

// Reconcile PersistentVolume
func (r *PersistentVolumeWatcher) Reconcile(_ context.Context, request reconcile.Request) (result reconcile.Result, err error) {
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

	r.Log = r.Log.WithValues("PersistentVolume", request.NamespacedName)
	r.Log.Info("reconciling PersistentVolume")

	pv := &corev1.PersistentVolume{}
	if err = r.Client.Get(context.TODO(), request.NamespacedName, pv); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("PersistentVolume not found")
			return result, nil
		}
		return result, err
	}

	if err = r.ensureStorageSystemLabel(pv); err != nil {
		switch err.(type) {
		case *util.UniqueFSCMatchError:
			r.Log.Info("cannot uniquely match FlashSystemCluster to PersistentVolume")
			return result, nil
		default:
			r.Log.Error(nil, "error while trying to match PersistentVolume to FlashSystemCluster")
			return result, err
		}
	}

	return result, nil
}

func (r *PersistentVolumeWatcher) ensureStorageSystemLabel(pv *corev1.PersistentVolume) error {
	pvMgmtAddr, err := r.getPVManagementAddress(pv)

	if err == nil {
		clustersByMgmtAddr, err := util.MapClustersByMgmtAddress(r.Client, r.Log)
		if err != nil {
			return err
		}

		pvLabels := pv.GetLabels()
		if pvLabels == nil {
			pvLabels = make(map[string]string)
		}

		fsc, fscExist := clustersByMgmtAddr[pvMgmtAddr]
		labelValue, labelExist := pvLabels[util.OdfFsStorageSystemLabelKey]
		if !fscExist {
			if !labelExist {
				return nil
			}
			r.Log.Info("cannot find FlashSystemCluster for PersistentVolume, removing StorageSystem label")
			delete(pvLabels, util.OdfFsStorageSystemLabelKey)
		} else {
			if labelExist && labelValue == fsc.Name {
				r.Log.Info("no change on PersistentVolume, skips adding StorageSystem label to PersistentVolume")
				return nil
			}
			r.Log.Info("adding StorageSystem label to PersistentVolume")
			pvLabels[util.OdfFsStorageSystemLabelKey] = fsc.Name
		}

		pv.SetLabels(pvLabels)
		if err = r.Client.Update(context.Background(), pv); err != nil {
			r.Log.Error(err, "failed to update PersistentVolume")
			return err
		}
		return nil
	}
	return err
}

func (r *PersistentVolumeWatcher) getPVManagementAddress(pv *corev1.PersistentVolume) (string, error) {
	r.Log.Info("looking for PersistentVolume management address")

	pvVolumeAttributes := pv.Spec.CSI.VolumeAttributes
	pvMgmtAddr, ok := pvVolumeAttributes[util.PVMgmtAddrKey]

	if !ok {
		r.Log.Info("looking for PersistentVolumeClaim")
		pvcRef := pv.Spec.ClaimRef
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      pvcRef.Name,
			Namespace: pvcRef.Namespace},
			pvc)
		if err != nil {
			r.Log.Error(err, "failed to get PersistentVolumeClaim")
			return "", err
		}

		r.Log.Info("looking for StorageClass")
		sc := &storagev1.StorageClass{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: *(pvc.Spec.StorageClassName)}, sc)
		if err != nil {
			r.Log.Error(err, "failed to get StorageClass")
			return "", err
		}

		_, isTopology := sc.Parameters[util.TopologyStorageClassByMgmtId]
		if isTopology {
			return "", &util.UniqueFSCMatchError{}
		}

		r.Log.Info("looking for StorageClass Secret")
		secret, err := util.GetStorageClassSecret(r.Client, r.Log, sc)
		if err != nil {
			r.Log.Error(err, "failed to get StorageClass Secret")
			return "", err
		}

		pvMgmtAddr = string(secret.Data[util.SecretManagementAddressKey])
	}
	return pvMgmtAddr, nil
}
