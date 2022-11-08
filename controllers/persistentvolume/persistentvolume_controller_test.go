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
	_ "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("PersistentVolume Controller", func() {
	const (
		PersistentVolumeName = "test-persistent-volume"
		namespace            = "openshift-storage"
		PersistenVolumeClaim = "test-persistent-volume-claim"

		timeout = time.Second * 20
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a new PersistentVolume", func() {
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

		It("Should create the PersistentVolume successfully", func() {
			By("By creating a new PersistentVolume")
			ctx := context.Background()
			persistentvolume := &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PersistentVolumeName,
					Namespace: namespace,
				},
				Spec: corev1.PersistentVolumeSpec{
					Capacity: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("512Gi"),
					},
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					ClaimRef: &corev1.ObjectReference{
						Kind:      util.PersistentVolumeClaimKind,
						Name:      PersistenVolumeClaim,
						Namespace: namespace,
					},
				},
			}
			Expect(k8sClient.Create(ctx, persistentvolume)).Should(Succeed())

			By("By checking the PersistentVolume is created successfully")
			pvLookupKey := types.NamespacedName{Name: PersistentVolumeName, Namespace: namespace}
			createdPV := &corev1.PersistentVolume{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pvLookupKey, createdPV)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdPV.Spec.ClaimRef.Kind).Should(Equal(util.PersistentVolumeClaimKind))
			Expect(createdPV.Spec.ClaimRef.Name).Should(Equal(PersistenVolumeClaim))
			Expect(createdPV.Spec.ClaimRef.Namespace).Should(Equal(namespace))

		})

		It("Should delete the PersistentVolume successfully", func() {
			By("By deleting the PersistentVolume")
			ctx := context.Background()
			pvLookupKey := types.NamespacedName{Name: PersistentVolumeName, Namespace: namespace}
			createdPV := &corev1.PersistentVolume{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pvLookupKey, createdPV)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(k8sClient.Delete(ctx, createdPV)).Should(Succeed())

			By("By checking the PersistentVolume is deleted successfully")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, pvLookupKey, createdPV)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeFalse())
		})
	})
})
