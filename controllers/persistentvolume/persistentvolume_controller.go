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
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

func reconcilePV(obj runtime.Object) bool {
	pv, ok := obj.(*corev1.PersistentVolume)
	if !ok {
		return false
	}
	return pv.Spec.CSI.Driver == util.CsiIBMBlockDriver && pv.Spec.ClaimRef != nil &&
		pv.Spec.ClaimRef.Kind == util.PersistentVolumeClaimKind
}

var pvPredicate = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		return reconcilePV(e.Object)
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		return reconcilePV(e.Object)
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		return reconcilePV(e.ObjectNew)
	},
	GenericFunc: func(e event.GenericEvent) bool {
		return reconcilePV(e.Object)
	},
}

type PersistentVolumeWatcher struct {
	Client    client.Client
	Scheme    *runtime.Scheme
	Log       logr.Logger
	Namespace string
}

func (r *PersistentVolumeWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.PersistentVolume{}, builder.WithPredicates(pvPredicate)).
		Complete(r)
}

// +kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;update;patch

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

	if err = r.addStorageSystemLabelToPV(pv); err != nil {
		return result, err
	}

	return result, nil
}

func (r *PersistentVolumeWatcher) addStorageSystemLabelToPV(pv *corev1.PersistentVolume) error {
	pvMgmtAddr, err := r.getPVManagementAddress(pv)

	if err != nil {
		clustersByMgmtAddr, err := util.MapClustersByMgmtAddress(r.Client, r.Log)
		if err != nil {
			return err
		}

		fsc, exist := clustersByMgmtAddr[pvMgmtAddr]
		if !exist {
			errMsg := "cannot find FlashSystemCluster for PersistentVolume"
			r.Log.Error(nil, errMsg)
			return fmt.Errorf(errMsg)
		}

		pvLabels := pv.GetLabels()
		if pvLabels == nil {
			pvLabels = make(map[string]string)
		}

		pvLabels[util.OdfFsStorageSystemLabelKey] = fsc.Name
		pv.SetLabels(pvLabels)

		r.Log.Info("adding StorageSystem label for PersistentVolume")
		if err = r.Client.Update(context.Background(), pv); err != nil {
			r.Log.Error(err, "failed to update PersistentVolume")
			return err
		}
		return nil
	}
	return err
}

func (r *PersistentVolumeWatcher) getPVManagementAddress(pv *corev1.PersistentVolume) (string, error) {
	pvVolumeAttributes := pv.Spec.CSI.VolumeAttributes
	pvMgmtAddr, ok := pvVolumeAttributes[util.PVMgmtAddrKey]

	if !ok {
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      pv.Spec.ClaimRef.Name,
			Namespace: pv.Spec.ClaimRef.Namespace},
			pvc)
		if err != nil {
			return "", err
		}

		sc := &storagev1.StorageClass{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: *(pvc.Spec.StorageClassName)}, sc)
		if err != nil {
			return "", err
		}

		_, isTopology := sc.Parameters[util.TopologyStorageClassByMgmtId]
		if isTopology {
			errMsg := "find multiple FlashSystemClusters for PersistentVolume"
			r.Log.Error(nil, errMsg)
			return "", fmt.Errorf(errMsg)
		}
		secret, err := util.GetStorageClassSecret(r.Client, r.Log, sc)
		if err != nil {
			return "", err
		}
		pvMgmtAddr = string(secret.Data[util.SecretManagementAddressKey])
	}
	return pvMgmtAddr, nil
}
