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
	"context"
	"encoding/json"
	"fmt"
	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"time"

	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	//	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("StorageClassWatcher", func() {
	const (
		FlashSystemName  = "flashsystemcluster-sample"
		namespace        = "openshift-storage"
		secretName       = "fs-secret-sample"
		storageClassName = "odf-flashsystemcluster"
		poolName         = "Pool0"
		fsType           = "ext4"
		volPrefix        = "product"
		spaceEff         = "thin"

		timeout = time.Second * 10
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("when creating StorageClass with provisioner is IBM block csi", func() {
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

		It("should create StorageClass successfully", func() {
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

			By("By creating a new flashsystemcluster")
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

			By("By creating a new StorageClass")
			sc := &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      storageClassName,
					Namespace: "",
				},
				Provisioner: util.CsiIBMBlockDriver,
				Parameters: map[string]string{
					"SpaceEfficiency": spaceEff,
					"pool":            poolName,
					"csi.storage.k8s.io/provisioner-secret-name":             sec.Name,
					"csi.storage.k8s.io/provisioner-secret-namespace":        sec.Namespace,
					"csi.storage.k8s.io/controller-publish-secret-name":      sec.Name,
					"csi.storage.k8s.io/controller-publish-secret-namespace": sec.Namespace,
					"csi.storage.k8s.io/controller-expand-secret-name":       sec.Name,
					"csi.storage.k8s.io/controller-expand-secret-namespace":  sec.Namespace,
					"csi.storage.k8s.io/fstype":                              fsType,
					"volume_name_prefix":                                     volPrefix,
				},
			}

			Expect(k8sClient.Create(ctx, sc)).Should(Succeed())

			By("By querying the created StorageClass")
			scLookupKey := types.NamespacedName{
				Name:      storageClassName,
				Namespace: "",
			}
			createdSc := &storagev1.StorageClass{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, scLookupKey, createdSc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By querying the ConfigMap")
			cmLookupKey := types.NamespacedName{
				Name:      util.PoolConfigmapName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			var sp util.FlashSystemClusterMapContent
			err := json.Unmarshal([]byte(createdCm.Data[FlashSystemName]), &sp)
			Expect(err).ToNot(HaveOccurred())
			Expect(sp.ScPoolMap[storageClassName]).Should(Equal(poolName))
			fmt.Println("created ConfigMap: ", createdCm, sp)

			By("By deleting StorageClass")
			Expect(k8sClient.Delete(ctx, createdSc)).Should(Succeed())

			createdSc = &storagev1.StorageClass{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, scLookupKey, createdSc)
				if err != nil {
					if errors.IsNotFound(err) {
						return true
					} else {
						return false
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			By("By querying the ConfigMap")
			createdCm = &corev1.ConfigMap{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err != nil {
					return false
				}

				var sp util.FlashSystemClusterMapContent
				err = json.Unmarshal([]byte(createdCm.Data[FlashSystemName]), &sp)
				Expect(err).ToNot(HaveOccurred())

				_, ok := sp.ScPoolMap[storageClassName]
				return !ok
			}, timeout, interval).Should(BeTrue())
		})

	})
})
