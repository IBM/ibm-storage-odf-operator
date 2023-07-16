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
	"encoding/json"
	"reflect"

	"time"

	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("FlashSystemClusterReconciler", func() {
	const (
		FlashSystemName  = "flashsystemcluster-sample"
		namespace        = "openshift-storage"
		secretName       = "fs-secret-sample"
		podName          = "ibm-storage-odf-operator"
		storageClassName = "odf-flashsystemcluster-sample"
		poolName         = "Pool0"
		fsType           = "ext4"
		volPrefix        = "odf"
		spaceEff         = "thick"

		timeout  = time.Second * 30
		interval = time.Millisecond * 400
	)

	Context("when creating FlashSystemCluster CR", func() {
		It("should create namespace successfully", func() {
			By("By creating a new namespace")
			ctx := context.TODO()
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}

			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			By("By querying the created namespace")
			nsLookupKey := types.NamespacedName{
				Name: namespace,
			}
			createdNs := &corev1.Namespace{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, nsLookupKey, createdNs)
				return err == nil
			}, timeout, interval).Should(BeTrue())

		})
		It("should create operator pod successfully", func() {
			By("By creating a new pod")
			ctx := context.TODO()

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  podName,
							Image: "nginx",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, pod)).Should(Succeed())

			By("By querying the created pod")
			podLookupKey := types.NamespacedName{
				Name:      podName,
				Namespace: namespace,
			}
			createdPod := &corev1.Pod{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, podLookupKey, createdPod)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("should create secret successfully", func() {
			By("By creating a new secret")
			ctx := context.TODO()
			sec := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"management_address": []byte("OS4xMTAuNzAuOTY="),
					"password":           []byte("ZnNkcml2ZXI="),
					"username":           []byte("ZnNkcml2ZXI="),
				},
			}

			Expect(k8sClient.Create(ctx, sec)).Should(Succeed())

			By("By querying the created Secret")
			secLookupKey := types.NamespacedName{
				Name:      secretName,
				Namespace: namespace,
			}
			createdSec := &corev1.Secret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secLookupKey, createdSec)
				return err == nil
			}, timeout, interval).Should(BeTrue())

		})

		It("should create CSI operator CR successfully", func() {
			instance := &odfv1alpha1.FlashSystemCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FlashSystemName,
					Namespace: namespace,
				},
				Spec: odfv1alpha1.FlashSystemClusterSpec{
					Name: FlashSystemName,
					Secret: corev1.SecretReference{
						Name:      secretName,
						Namespace: namespace,
					},
				},
			}
			err := testFlashSystemClusterReconciler.ensureFlashSystemCSICR(instance)
			Expect(err).ToNot(HaveOccurred())

			namespaces, err := GetAllNamespace(testFlashSystemClusterReconciler.Config)
			Expect(err).ToNot(HaveOccurred())

			isCSICRFound, err := HasIBMBlockCSICRExisted(namespaces, testFlashSystemClusterReconciler.CSIDynamicClient)
			Expect(err).ToNot(HaveOccurred())

			By("expecting submitted")
			Expect(isCSICRFound).Should(BeTrue())
		})

		It("should create FlashSystemCluster successfully", func() {
			By("By creating a new FlashSystemCluster")
			ctx := context.TODO()
			instance := &odfv1alpha1.FlashSystemCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FlashSystemName,
					Namespace: namespace,
				},
				Spec: odfv1alpha1.FlashSystemClusterSpec{
					Name: FlashSystemName,
					Secret: corev1.SecretReference{
						Name:      secretName,
						Namespace: namespace,
					},
					InsecureSkipVerify: true,
					DefaultPool: &odfv1alpha1.StorageClassConfig{
						StorageClassName: storageClassName,
						PoolName:         poolName,
						FsType:           fsType,
						VolumeNamePrefix: volPrefix,
						SpaceEfficiency:  spaceEff,
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			fsLookupKey := types.NamespacedName{
				Name:      FlashSystemName,
				Namespace: namespace,
			}
			createdFs := &odfv1alpha1.FlashSystemCluster{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, fsLookupKey, createdFs)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			cmLookupKey := types.NamespacedName{
				Name:      util.FscCmName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err == nil {
					var sp util.FscConfigMapFscContent
					err = json.Unmarshal([]byte(createdCm.Data[FlashSystemName]), &sp)
					if err == nil {
						return sp.Secret == secretName
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			createdDeployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{Name: fsDeploymentName, Namespace: instance.Namespace},
					createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			createdSec := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      secretName,
						Namespace: namespace,
					},
					createdSec)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			exporterImg, err := util.GetExporterImage()
			Expect(err).ToNot(HaveOccurred())

			expectedDeployment, err := InitExporterDeployment(instance, corev1.PullIfNotPresent, exporterImg)
			Expect(err).ToNot(HaveOccurred())

			// TODO: customize deep comparison
			isSame := reflect.DeepEqual(createdDeployment.Name, expectedDeployment.Name)
			Expect(isSame).Should(BeTrue())

			createdService := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      fsServiceName,
						Namespace: namespace,
					},
					createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			createdServiceMonitor := &monitoringv1.ServiceMonitor{}
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      fsServiceMonitorName,
						Namespace: namespace,
					},
					createdServiceMonitor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			scLookupKey := types.NamespacedName{
				Name:      storageClassName,
				Namespace: namespace,
			}
			createdSc := &storagev1.StorageClass{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, scLookupKey, createdSc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			createdPromRule := &monitoringv1.PrometheusRule{}
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      instance.Name,
						Namespace: namespace,
					},
					createdPromRule)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			currentFS := &odfv1alpha1.FlashSystemCluster{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, fsLookupKey, currentFS)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			isNotReady := currentFS.Status.Phase == util.PhaseNotReady
			Expect(isNotReady).Should(BeTrue())

			util.SetReconcileCompleteCondition(&currentFS.Status.Conditions, odfv1alpha1.ReasonReconcileCompleted, "reconciling done")

			// simulate driver ready
			util.SetStatusCondition(&currentFS.Status.Conditions, odfv1alpha1.Condition{
				Type:   odfv1alpha1.ExporterReady,
				Status: corev1.ConditionTrue,
			})

			util.SetStatusCondition(&currentFS.Status.Conditions, odfv1alpha1.Condition{
				Type:   odfv1alpha1.StorageClusterReady,
				Status: corev1.ConditionTrue,
			})

			err = k8sClient.Status().Update(context.TODO(), currentFS)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, fsLookupKey, currentFS)
				return currentFS.Status.Phase == util.PhaseReady && err == nil
			}, timeout, interval).Should(BeTrue())

		})

		It("should delete FlashSystemCluster successfully", func() {
			By("By deleting a FlashSystemCluster")

			ctx := context.TODO()
			fsLookupKey := types.NamespacedName{
				Name:      FlashSystemName,
				Namespace: namespace,
			}
			createdFs := &odfv1alpha1.FlashSystemCluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, fsLookupKey, createdFs)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, createdFs)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, fsLookupKey, createdFs)
				return err != nil
			}, timeout, interval).Should(BeTrue())

			cmLookupKey := types.NamespacedName{
				Name:      util.FscCmName,
				Namespace: namespace,
			}

			createdCm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err == nil {
					_, ok := createdCm.Data[FlashSystemName]
					return ok == false
				}
				return false
			}, timeout, interval).Should(BeTrue())

		})
	})
})
