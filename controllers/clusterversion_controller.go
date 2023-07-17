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
	configv1 "github.com/openshift/api/config/v1"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/IBM/ibm-storage-odf-operator/console"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
)

// ClusterVersionReconciler reconciles a ClusterVersion object
type ClusterVersionReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	ConsolePort int
}

//+kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions/finalizers,verbs=update
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=deployments/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=*

// Reconcile - For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ClusterVersionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	instance := configv1.ClusterVersion{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, &instance); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureConsolePlugin(instance.Status.Desired.Version); err != nil {
		logger.Error(err, "could not ensure compatibility for IBM consolePlugin")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.Add(manager.RunnableFunc(func(context.Context) error {
		clusterVersion, err := util.DetermineOpenShiftVersion(r.Client)
		if err != nil {
			return err
		}

		return r.ensureConsolePlugin(clusterVersion)
	}))
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1.ClusterVersion{}).
		Watches(&source.Kind{
			Type: &consolev1alpha1.ConsolePlugin{},
		}, &handler.EnqueueRequestForOwner{OwnerType: &corev1.Pod{}}).
		Complete(r)
}

func (r *ClusterVersionReconciler) ensureConsolePlugin(clusterVersion string) error {
	// The base path to where the request are sent
	basePath := console.GetBasePath(clusterVersion)

	// Get IBM console Deployment
	ibmConsoleDeployment := console.GetDeployment(watchNamespace)
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      ibmConsoleDeployment.Name,
		Namespace: ibmConsoleDeployment.Namespace,
	}, ibmConsoleDeployment)
	if err != nil {
		return err
	}

	// Create/Update IBM console Service
	ibmConsoleService := console.GetService(r.ConsolePort, watchNamespace)
	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, ibmConsoleService, func() error {
		return controllerutil.SetControllerReference(ibmConsoleDeployment, ibmConsoleService, r.Scheme)
	})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// Create/Update IBM console ConsolePlugin
	ibmConsolePlugin := console.GetConsolePluginCR(r.ConsolePort, basePath, watchNamespace)

	consolePluginOwnerDetails, err := getOperatorPodOwnerDetails(ibmConsoleDeployment.Namespace, r.Client)
	if err != nil {
		return err
	}
	if !IsOwnerExist(ibmConsolePlugin.GetOwnerReferences(), consolePluginOwnerDetails) {
		ibmConsolePlugin.SetOwnerReferences(append(ibmConsolePlugin.GetOwnerReferences(), consolePluginOwnerDetails))
	}

	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, ibmConsolePlugin, func() error {
		return nil
	})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
