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

package controllers

import (
	"context"
	"fmt"
	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/IBM/ibm-storage-odf-operator/controllers/storageclass"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var flashSystemClusterFinalizer = "flashsystemcluster.odf.ibm.com"

// watchNamespace is the namespace the operator is watching.
var (
	watchNamespace string
)

type SecretMapper struct {
	reconciler *FlashSystemClusterReconciler
}

func (s *SecretMapper) SecretToClusterMapFunc(object client.Object) []reconcile.Request {
	clusters := &odfv1alpha1.FlashSystemClusterList{}

	err := s.reconciler.Client.List(context.TODO(), clusters)
	if err != nil {
		s.reconciler.Log.Error(err, "failed to list flashsystemcluster", "SecretMapper", s)
		return nil
	}

	requests := []reconcile.Request{}
	for _, c := range clusters.Items {
		if c.Spec.Secret.Name == object.GetName() &&
			c.Spec.Secret.Namespace == object.GetNamespace() {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: c.GetNamespace(),
					Name:      c.GetName(),
				},
			}
			requests = append(requests, req)
		}

	}

	if len(requests) > 0 {
		s.reconciler.Log.Info("reflect secret update to flashsystemcluster instance", "SecretMapper", requests)
	}

	return requests
}

// FlashSystemClusterReconciler reconciles a FlashSystemCluster object
type FlashSystemClusterReconciler struct {
	client.Client
	CSIDynamicClient dynamic.NamespaceableResourceInterface
	Config           *rest.Config
	Log              logr.Logger
	Scheme           *runtime.Scheme
	ExporterImage    string
	IsCSICRCreated   bool
}

//+kubebuilder:rbac:groups=odf.ibm.com,resources=flashsystemclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=odf.ibm.com,resources=flashsystemclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=odf.ibm.com,resources=flashsystemclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=csi.ibm.com,resources=ibmblockcsis,verbs=get;list;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;prometheusrules,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FlashSystemCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *FlashSystemClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	result := reconcile.Result{}
	prevLogger := r.Log

	defer func() {
		r.Log = prevLogger
		if errors.IsConflict(err) {
			r.Log.Info("requeue for resource update conflicts")
			// update return values
			result = reconcile.Result{Requeue: true}
			err = nil
		}
	}()

	r.Log = r.Log.WithValues("flashsystemcluster", req.NamespacedName)
	r.Log.Info("Reconciling FlashSystemCluster")

	instance := &odfv1alpha1.FlashSystemCluster{}
	err = r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("FlashSystemCluster resource was not found")
			return result, nil
		}
		// Error reading the object - requeue the request.
		return result, err
	}

	result, err = r.reconcile(instance)

	statusError := r.Client.Status().Update(context.TODO(), instance)

	if err != nil {
		r.Log.Error(err, "failed to reconcile")
		return result, err
	} else if statusError != nil {
		r.Log.Error(statusError, "failed to update conditions to status")
		return result, statusError
	} else {
		return result, nil
	}
}

func (r *FlashSystemClusterReconciler) reconcile(instance *odfv1alpha1.FlashSystemCluster) (ctrl.Result, error) {
	var err error

	// Check GetDeletionTimestamp to determine if the object is under deletion
	if instance.GetDeletionTimestamp().IsZero() {
		if !util.IsContain(instance.GetFinalizers(), flashSystemClusterFinalizer) {
			r.Log.Info("Append flashsystemcluster to finalizer")
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, flashSystemClusterFinalizer)
			if err := r.Client.Update(context.TODO(), instance); err != nil {
				r.Log.Info("Update Error", "MetaUpdateErr", "Failed to update FlashSystemCluster with finalizer")
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is marked for deletion
		if util.IsContain(instance.GetFinalizers(), flashSystemClusterFinalizer) {
			r.Log.Info("Removing finalizer")
			if err := r.deleteDefaultStorageClass(instance); err != nil {
				return reconcile.Result{}, err
			}

			// Once all finalizers have been removed, the object will be deleted
			instance.ObjectMeta.Finalizers = util.Remove(instance.ObjectMeta.Finalizers, flashSystemClusterFinalizer)
			if err := r.Client.Update(context.TODO(), instance); err != nil {
				r.Log.Info("Update Error", "MetaUpdateErr", "Failed to remove finalizer from FlashSystemCluster")
				return reconcile.Result{}, err
			}
		}
		r.Log.Info("Object is terminated, skipping reconciliation")
		return reconcile.Result{}, nil
	}
	// Reading the secret, if it has ownership then skip, else update the secret with details of the FlashSystemCluster
	secret := &corev1.Secret{}
	err = r.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: instance.Spec.Secret.Name, Namespace: instance.Spec.Secret.Namespace},
		secret)

	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Secret not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if secret.OwnerReferences == nil {
		r.Log.Info("FlashSystemCluster Secret does not have an owner reference, adding it now.")
		// Creating an OwnerReference object with the info of the FlashSystemCluster for it to be the owner of the secret
		newOwnerForSecret := v1.OwnerReference{
			Name:       instance.Name,
			Kind:       instance.Kind,
			APIVersion: instance.APIVersion,
			UID:        instance.UID,
		}
		secret.SetOwnerReferences([]v1.OwnerReference{newOwnerForSecret})
		err := r.Client.Update(context.TODO(), secret)
		if err != nil {
			r.Log.Error(err, "Update Error: Failed to update secret with owner reference")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	r.Log.Info("step: create or check FlashSystem CSI CR")
	err = r.ensureFlashSystemCSICR(instance)
	if err != nil {
		r.Log.Error(err, "failed to ensureFlashSystemCSICR")
		return reconcile.Result{}, err
	}

	r.Log.Info("step: reset progressing conditions of the FlashSystemCluster resource")
	if util.IsStatusConditionFalse(instance.Status.Conditions, odfv1alpha1.ConditionProgressing) {
		reason := odfv1alpha1.ReasonReconcileInit
		message := "processing flashsystem ODF resources"
		util.SetReconcileProgressingCondition(&instance.Status.Conditions, reason, message)
		instance.Status.Phase = util.PhaseProgressing
	}

	r.Log.Info("step: ensureScPoolConfigMap")
	if err = r.ensureScPoolConfigMap(); err != nil {
		reason := odfv1alpha1.ReasonReconcileFailed
		message := fmt.Sprintf("failed to ensureScPoolConfigMap: %v", err)
		util.SetReconcileErrorCondition(&instance.Status.Conditions, reason, message)
		instance.Status.Phase = util.PhaseError

		return reconcile.Result{}, err
	}
	newOwnerDetails := v1.OwnerReference{
		Name:       instance.Name,
		Kind:       instance.Kind,
		APIVersion: instance.APIVersion,
		UID:        instance.UID,
	}

	r.Log.Info("step: ensureExporterService")
	if err = r.ensureExporterService(instance, newOwnerDetails); err != nil {
		reason := odfv1alpha1.ReasonReconcileFailed
		message := fmt.Sprintf("failed to ensureExporterService: %v", err)
		util.SetReconcileErrorCondition(&instance.Status.Conditions, reason, message)
		instance.Status.Phase = util.PhaseError

		r.createEvent(instance, corev1.EventTypeWarning,
			util.FailedCreateServiceReason, message)

		return reconcile.Result{}, err
	}

	r.Log.Info("step: ensureExporterDeployment")
	if err = r.ensureExporterDeployment(instance, newOwnerDetails); err != nil {
		reason := odfv1alpha1.ReasonReconcileFailed
		message := fmt.Sprintf("failed to ensureExporterDeployment: %v", err)
		util.SetReconcileErrorCondition(&instance.Status.Conditions, reason, message)
		instance.Status.Phase = util.PhaseError

		r.createEvent(instance, corev1.EventTypeWarning,
			util.FailedLaunchBlockExporterReason, message)

		return reconcile.Result{}, err
	}

	r.Log.Info("step: ensureExporterServiceMonitor")
	if err = r.ensureExporterServiceMonitor(instance, newOwnerDetails); err != nil {
		reason := odfv1alpha1.ReasonReconcileFailed
		message := fmt.Sprintf("failed to ensureExporterServiceMonitor: %v", err)
		util.SetReconcileErrorCondition(&instance.Status.Conditions, reason, message)
		instance.Status.Phase = util.PhaseError

		r.createEvent(instance, corev1.EventTypeWarning,
			util.FailedCreateServiceMonitorReason, message)

		return reconcile.Result{}, err
	}

	util.SetStatusCondition(&instance.Status.Conditions, odfv1alpha1.Condition{
		Type:   odfv1alpha1.ExporterCreated,
		Status: corev1.ConditionTrue,
	})

	r.Log.Info("step: ensureDefaultStorageClass")
	if err = r.ensureDefaultStorageClass(instance); err != nil {
		reason := odfv1alpha1.ReasonReconcileFailed
		message := fmt.Sprintf("failed to ensureDefaultStorageClass: %v", err)
		util.SetReconcileErrorCondition(&instance.Status.Conditions, reason, message)
		instance.Status.Phase = util.PhaseError

		r.createEvent(instance, corev1.EventTypeWarning,
			util.FailedCreateStorageClassReason, message)

		return reconcile.Result{}, err
	}

	r.Log.Info("step: enablePrometheusRules")
	if err = r.enablePrometheusRules(instance); err != nil {
		reason := odfv1alpha1.ReasonReconcileFailed
		message := fmt.Sprintf("failed to enablePrometheusRules: %v", err)
		util.SetReconcileErrorCondition(&instance.Status.Conditions, reason, message)
		instance.Status.Phase = util.PhaseError

		r.createEvent(instance, corev1.EventTypeWarning,
			util.FailedCreatePromRuleReason, message)

		return reconcile.Result{}, err
	}

	util.SetReconcileCompleteCondition(&instance.Status.Conditions, odfv1alpha1.ReasonReconcileCompleted, "reconciling done")

	r.Log.Info("step: check and update status phase")
	if util.IsStatusConditionTrue(instance.Status.Conditions, odfv1alpha1.ConditionReconcileComplete) &&
		util.IsStatusConditionTrue(instance.Status.Conditions, odfv1alpha1.ExporterReady) &&
		util.IsStatusConditionTrue(instance.Status.Conditions, odfv1alpha1.StorageClusterReady) &&
		util.IsStatusConditionTrue(instance.Status.Conditions, odfv1alpha1.ProvisionerReady) {
		instance.Status.Phase = util.PhaseReady
	} else if util.IsStatusConditionTrue(instance.Status.Conditions, odfv1alpha1.ConditionProgressing) {
		instance.Status.Phase = util.PhaseProgressing
	} else {
		instance.Status.Phase = util.PhaseNotReady
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlashSystemClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ns, err := util.GetWatchNamespace()
	if err != nil {
		return err
	}
	watchNamespace = ns

	exporterImage, err := util.GetExporterImage()
	if err != nil {
		return err
	}
	r.ExporterImage = exporterImage

	secretMapper := &SecretMapper{
		reconciler: r,
	}

	//TODO: it seems operator-sdk 1.5 + golang 1.5 fails to watch resources through Owns
	return ctrl.NewControllerManagedBy(mgr).
		For(&odfv1alpha1.FlashSystemCluster{}).
		Watches(&source.Kind{
			Type: &appsv1.Deployment{},
		}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{
			Type: &corev1.Service{},
		}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{
			Type: &monitoringv1.ServiceMonitor{},
		}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{
			Type: &corev1.Secret{},
		}, handler.EnqueueRequestsFromMapFunc(secretMapper.SecretToClusterMapFunc)).
		Complete(r)

}

func (r *FlashSystemClusterReconciler) createEvent(instance *odfv1alpha1.FlashSystemCluster, eventtype, reason, message string) {
	r.Log.Info(message)

	event := util.InitK8sEvent(instance, eventtype, reason, message)
	err := r.Client.Create(context.TODO(), event)
	if err != nil {
		r.Log.Error(err, "failed to create Event", "reason", reason, "message", message)
	}
}

// this object will not bind with instance
func (r *FlashSystemClusterReconciler) ensureScPoolConfigMap() error {
	expectedScPoolConfigMap := storageclass.InitScPoolConfigMap(watchNamespace)
	foundScPoolConfigMap := &corev1.ConfigMap{}

	err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: util.PoolConfigmapName, Namespace: watchNamespace},
		foundScPoolConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("create StorageClassPool ConfigMap")
			return r.Client.Create(context.TODO(), expectedScPoolConfigMap)
		}

		r.Log.Error(err, "failed to create StorageClassPool ConfigMap")
		return err
	}

	// storageclass_controller will update it.

	return nil
}

func (r *FlashSystemClusterReconciler) ensureExporterDeployment(instance *odfv1alpha1.FlashSystemCluster, newOwnerDetails v1.OwnerReference) error {

	exporterImg, err := util.GetExporterImage()
	if err != nil {
		r.Log.Error(err, "failed to get exporter image from pod env variable")
	}

	deploymentName := getExporterDeploymentName()

	foundSecret := &corev1.Secret{}
	err = r.Client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.Secret.Name,
			Namespace: instance.Spec.Secret.Namespace,
		}, foundSecret)

	if err != nil {
		return err
	}

	expectedDeployment, err := InitExporterDeployment(instance, corev1.PullIfNotPresent, exporterImg)
	if err != nil {
		return err
	}

	foundDeployment := &appsv1.Deployment{}
	err = r.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: deploymentName, Namespace: instance.Namespace},
		foundDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("create exporter deployment")
			return r.Client.Create(context.TODO(), expectedDeployment)
		}

		r.Log.Error(err, "failed to create exporter deployment")
		return err
	}

	updatedDeployment := updateExporterDeployment(foundDeployment, expectedDeployment, newOwnerDetails)
	if updatedDeployment != nil {
		r.Log.Info("update exporter deployment")
		return r.Client.Update(context.TODO(), updatedDeployment)
	}

	r.Log.Info("existing exporter deployment is expected with no change")
	return nil
}

func (r *FlashSystemClusterReconciler) ensureExporterService(instance *odfv1alpha1.FlashSystemCluster, newOwnerDetails v1.OwnerReference) error {

	expectedService := InitExporterMetricsService(instance)
	serviceName := getExporterMetricsServiceName()
	foundService := &corev1.Service{}

	err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: serviceName, Namespace: instance.Namespace},
		foundService)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("create exporter service")
			return r.Client.Create(context.TODO(), expectedService)
		}

		r.Log.Error(err, "failed to create exporter service")
		return err
	}

	updatedService := updateExporterMetricsService(foundService, expectedService, newOwnerDetails)
	if updatedService != nil {
		r.Log.Info("update exporter service")
		return r.Client.Update(context.TODO(), updatedService)
	}

	r.Log.Info("existing exporter service is expected with no change")
	return nil
}

func (r *FlashSystemClusterReconciler) ensureExporterServiceMonitor(instance *odfv1alpha1.FlashSystemCluster, newOwnerDetails v1.OwnerReference) error {
	expectedServiceMonitor := InitExporterMetricsServiceMonitor(instance)
	serviceMonitorName := getExporterMetricsServiceMonitorName()
	foundServiceMonitor := &monitoringv1.ServiceMonitor{}

	err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: serviceMonitorName, Namespace: instance.Namespace},
		foundServiceMonitor)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("create exporter servicemonitor")
			return r.Client.Create(context.TODO(), expectedServiceMonitor)
		}

		r.Log.Error(err, "failed to get exporter servicemonitor")
		return err
	}

	updatedServiceMonitor := updateExporterMetricsServiceMonitor(foundServiceMonitor, expectedServiceMonitor, newOwnerDetails)
	if updatedServiceMonitor != nil {
		r.Log.Info("update exporter servicemonitor")
		return r.Client.Update(context.TODO(), updatedServiceMonitor)
	}

	r.Log.Info("existing exporter servicemonitor is expected with no change")
	return nil
}

// create storage class in case of pool parameters provided.
func (r *FlashSystemClusterReconciler) ensureDefaultStorageClass(instance *odfv1alpha1.FlashSystemCluster) error {
	if instance.Spec.DefaultPool == nil {
		return nil
	}

	var found bool
	expectedStorageClass := InitDefaultStorageClass(instance)
	foundStorageClass := &storagev1.StorageClass{}

	err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: instance.Spec.DefaultPool.StorageClassName},
		foundStorageClass)

	if err != nil {
		if errors.IsNotFound(err) {
			found = false
		} else {
			r.Log.Error(err, "failed to get default StorageClass")
			return err
		}
	} else {
		found = true
	}

	if found {
		if compareDefaultStorageClass(foundStorageClass, expectedStorageClass) {
			r.Log.Info("no change on default StorageClass, skip reconciling")
			return nil
		}

		err = r.Client.Delete(
			context.TODO(),
			foundStorageClass)
		if err != nil {
			r.Log.Error(err, "failed to delete default StorageClass")
			return err
		}

		r.createEvent(instance, corev1.EventTypeWarning,
			util.DeletedDuplicatedStorageClassReason, "delete StorageClass with same name as default StorageClass")
	}

	r.Log.Info("create default StorageClass")
	return r.Client.Create(context.TODO(), expectedStorageClass)
}

func (r *FlashSystemClusterReconciler) deleteDefaultStorageClass(instance *odfv1alpha1.FlashSystemCluster) error {
	if instance.Spec.DefaultPool == nil {
		return nil
	}

	expectedStorageClass := InitDefaultStorageClass(instance)
	foundStorageClass := &storagev1.StorageClass{}

	err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: instance.Spec.DefaultPool.StorageClassName, Namespace: ""},
		foundStorageClass)
	if err != nil {
		if errors.IsNotFound(err) {
			// do nothing
			return nil
		}

		r.Log.Error(err, "failed to get default StorageClass")
		return err
	}

	if expectedStorageClass == nil {
		r.Log.Info("leave existing default StorageClass alone")
	} else {
		err = r.Client.Delete(
			context.TODO(),
			foundStorageClass)
		if err != nil {
			r.Log.Error(err, "failed to delete default StorageClass")
			return err
		} else {
			r.Log.Info("delete default StorageClass")
		}
	}

	return nil
}

func (r *FlashSystemClusterReconciler) enablePrometheusRules(instance *odfv1alpha1.FlashSystemCluster) error {
	rule, err := getPrometheusRules(instance)
	if err != nil {
		r.Log.Error(err, "prometheus rules file not found")
		return err
	}
	err = r.CreateOrUpdatePrometheusRules(rule)
	if err != nil {
		r.Log.Error(err, "unable to deploy Prometheus rules")
		return err
	}
	return nil
}

// CreateOrUpdatePrometheusRules creates or updates Prometheus Rule
func (r *FlashSystemClusterReconciler) CreateOrUpdatePrometheusRules(rule *monitoringv1.PrometheusRule) error {
	err := r.Client.Create(context.TODO(), rule)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			oldRule := &monitoringv1.PrometheusRule{}
			err = r.Client.Get(context.TODO(), types.NamespacedName{Name: rule.Name, Namespace: rule.Namespace}, oldRule)
			if err != nil {
				return fmt.Errorf("failed while fetching PrometheusRule: %v", err)
			}
			oldRule.Spec = rule.Spec
			err := r.Client.Update(context.TODO(), oldRule)
			if err != nil {
				return fmt.Errorf("failed while updating PrometheusRule: %v", err)
			}
		} else {
			return fmt.Errorf("failed while creating PrometheusRule: %v", err)
		}
	}
	return nil
}

func (r *FlashSystemClusterReconciler) ensureFlashSystemCSICR(instance *odfv1alpha1.FlashSystemCluster) error {
	if r.IsCSICRCreated {
		r.Log.Info("flashsystem CSI CR is already created, skip")
		return nil
	}

	// query if IBMBlockCSI CR has already existed in this cluster.
	namespaces, err := GetAllNamespace(r.Config)
	if err != nil {
		return err
	}

	isCSICRFound, err := HasIBMBlockCSICRExisted(namespaces, r.CSIDynamicClient)
	if err != nil {
		return err
	}

	if !isCSICRFound {
		// create CSI CR
		r.Log.Info("start to create CSI CR instance...")
		obj, err := CreateIBMBlockCSICR(r.CSIDynamicClient, instance.Namespace)
		if err != nil {
			r.createEvent(instance, corev1.EventTypeWarning,
				util.FailedLaunchBlockCSIReason,
				fmt.Sprintf("CSI CR:  %s/%s", obj.GetNamespace(), obj.GetName()))

			return err
		}
		r.createEvent(instance, corev1.EventTypeNormal,
			util.SuccessfulLaunchBlockCSIReason,
			fmt.Sprintf("CSI CR:  %s/%s", obj.GetNamespace(), obj.GetName()))
	} else {
		r.createEvent(instance, corev1.EventTypeNormal,
			util.SuccessfulDetectBlockCSIReason,
			"CSI CR is found, skip to create CSI CR instance")
	}

	r.IsCSICRCreated = true
	return nil
}
