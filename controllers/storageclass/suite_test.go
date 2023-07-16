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
	"fmt"
	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
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

	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var isRealCluster bool
var testStorageClassWatcher *StorageClassWatcher

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"StorageClass Suite",
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
				filepath.Join("..", "..", "config", "crd", "bases"),
			},
			ErrorIfCRDPathMissing: true,
		}
		fmt.Println("NOT Real k8s")
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

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
	err = os.Setenv(util.OperatorPodNameEnvVar, "ibm-storage-odf-operator")
	Expect(err).NotTo(HaveOccurred())

	// create manager and register controller for tset
	ns, err := util.GetWatchNamespace()
	Expect(err).ToNot(HaveOccurred())

	err = odfv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace: ns,
		Scheme:    scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	testStorageClassWatcher = &StorageClassWatcher{
		Client:    mgr.GetClient(),
		Namespace: ns,
		Log:       ctrl.Log.WithName("controllers").WithName("StorageClassWatcher"),
		Scheme:    mgr.GetScheme(),
	}
	err = testStorageClassWatcher.SetupWithManager(mgr)
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
