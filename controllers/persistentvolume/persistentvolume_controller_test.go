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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var k8sClient client.Client

var _ = Describe("PersistentVolume Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		PersistentVolumeName = "test-persistentvolume-name"
		namespace            = "openshift-storage"

		timeout = time.Second * 20
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a new PersistentVolume", func() {
		It("Should create the PersistentVolume successfully", func() {
			By("By creating a new PersistentVolume")
			ctx := context.Background()
			persistentvolume := &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PersistentVolumeName,
					Namespace: namespace,
				},
				Spec: corev1.PersistentVolumeSpec{
					ClaimRef: &corev1.ObjectReference{
						Kind: util.PersistentVolumeClaimKind,
					},
				},
			}
			Expect(k8sClient.Create(ctx, persistentvolume)).Should(Succeed())

			By("By checking the PersistentVolume is created successfully")
			persistentvolumeLookupKey := types.NamespacedName{Name: PersistentVolumeName, Namespace: namespace}
			createdPersistentVolume := &corev1.PersistentVolume{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, persistentvolumeLookupKey, createdPersistentVolume)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdPersistentVolume.Spec.ClaimRef.Kind).Should(Equal(util.PersistentVolumeClaimKind))

			By("By deleting the PersistentVolume")
			Expect(k8sClient.Delete(ctx, persistentvolume)).Should(Succeed())

			By("By checking the PersistentVolume is deleted successfully")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, persistentvolumeLookupKey, createdPersistentVolume)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeFalse())
		})

		// it should test the update of the PersistentVolume
		It("Should update the PersistentVolume successfully", func() {
			By("By creating a new PersistentVolume")
			ctx := context.Background()
			persistentvolume := &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PersistentVolumeName,
					Namespace: namespace,
				},
				Spec: corev1.PersistentVolumeSpec{
					ClaimRef: &corev1.ObjectReference{
						Kind: util.PersistentVolumeClaimKind,
					},
				},
			}
			Expect(k8sClient.Create(ctx, persistentvolume)).Should(Succeed())

			By("By checking the PersistentVolume is created successfully")
			persistentvolumeLookupKey := types.NamespacedName{Name: PersistentVolumeName, Namespace: namespace}
			createdPersistentVolume := &corev1.PersistentVolume{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, persistentvolumeLookupKey, createdPersistentVolume)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdPersistentVolume.Spec.ClaimRef.Kind).Should(Equal(util.PersistentVolumeClaimKind))

			By("By updating the PersistentVolume")
			createdPersistentVolume.Spec.ClaimRef.Kind = "new-kind-test"
			Expect(k8sClient.Update(ctx, createdPersistentVolume)).Should(Succeed())

			By("By checking the PersistentVolume is updated successfully")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, persistentvolumeLookupKey, createdPersistentVolume)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdPersistentVolume.Spec.ClaimRef.Kind).Should(Equal("new-kind-test"))

			By("By deleting the PersistentVolume")
			Expect(k8sClient.Delete(ctx, persistentvolume)).Should(Succeed())

			By("By checking the PersistentVolume is deleted successfully")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, persistentvolumeLookupKey, createdPersistentVolume)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeFalse())

		})
	})
})
