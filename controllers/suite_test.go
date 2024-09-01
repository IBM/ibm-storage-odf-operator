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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	odfv1 "github.com/IBM/ibm-storage-odf-operator/api/v1"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var isRealCluster bool
var testFlashSystemClusterReconciler *FlashSystemClusterReconciler

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

// by default testenv sets up local api server & etcd environment without external dependencies
// test method for real cluster environment
//  1. prepare the openshift environment with CRD & ns is created
//  2. export TEST_USE_EXISTING_CLUSTER=true
//  3. make test
var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// Prevent the metrics listener being created
	metrics.DefaultBindAddress = "0"

	By("bootstrapping test environment")

	isRealCluster = false
	if os.Getenv("TEST_USE_EXISTING_CLUSTER") == "true" {
		isRealCluster = true
	}

	if isRealCluster {
		t := true
		testEnv = &envtest.Environment{
			UseExistingCluster: &t,
		}
	} else {
		testEnv = &envtest.Environment{
			CRDDirectoryPaths: []string{
				filepath.Join("..", "config", "crd", "bases"),
				filepath.Join("..", "config", "samples", "csi-crds"),
				filepath.Join("..", "config", "samples", "servicemonitor-crds"),
				filepath.Join("..", "config", "samples", "prometheus-crds"),
			},
			ErrorIfCRDPathMissing: true,
		}
		fmt.Println("NOT Real k8s")
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = odfv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = monitoringv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	// NOTE: use k8sClient but not mgr.Client for read-after-write validation
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// set environment before test
	err = os.Setenv(util.WatchNamespaceEnvVar, "openshift-storage")
	Expect(err).NotTo(HaveOccurred())
	err = os.Setenv(util.ExporterImageEnvVar, "docker.io/ibmcom/ibm-storage-odf-block-driver:v0.0.22")
	Expect(err).NotTo(HaveOccurred())

	// create manager and register controller for test
	ns, err := util.GetWatchNamespace()
	Expect(err).ToNot(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace: ns,
		Scheme:    scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	dynamicClient, err := GetIBMBlockCSIDynamicClient(mgr.GetConfig())
	Expect(err).ToNot(HaveOccurred())

	testFlashSystemClusterReconciler = &FlashSystemClusterReconciler{
		Client:           mgr.GetClient(),
		CSIDynamicClient: dynamicClient,
		IsCSICRCreated:   false,
		Config:           mgr.GetConfig(),
		Log:              ctrl.Log.WithName("controllers").WithName("FlashSystemCluster"),
		Scheme:           mgr.GetScheme(),
	}

	err = testFlashSystemClusterReconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

}, 60)

var _ = AfterSuite(func() {
	fmt.Println("tearing down the test environment")
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())

	// Put the DefaultBindAddress back
	metrics.DefaultBindAddress = ":8080"
})
