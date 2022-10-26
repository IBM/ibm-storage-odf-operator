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
	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	//	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("StorageClassWatcher", func() {
	const (
		FlashSystemName          = "flashsystemcluster-sample"
		namespace                = "openshift-storage"
		secretName               = "fs-secret-sample"
		topologySecretName       = "topology-secret"
		storageClassName         = "odf-flashsystemcluster"
		topologyStorageClassName = "topology-storageclass"
		poolName                 = "Pool0"
		topologyPoolName         = "demo-pool-1"
		fsType                   = "ext4"
		volPrefix                = "product"
		spaceEff                 = "thin"
		byManagementIdData       = "{\"demo-management-id-1\":{\"pool\":\"demo-pool-1\",\"SpaceEfficiency\":\"dedup_compressed\",\"volume_name_prefix\":\"demo-prefix-1\"}}"
		topologySecretConfigData = "{\"demo-management-id-1\": {\"username\": \"testUsername\",\"testPassword\": \"password\",\"management_address\": \"9.110.70.96\"}}"

		timeout = time.Second * 20
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("when creating StorageClass with provisioner IBM block csi", func() {
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

			By("By creating a new secret")
			sec := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"management_address": []byte("9.110.70.96"),
					"password":           []byte("testUsername"),
					"username":           []byte("testPassword"),
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

			By("By creating a new FlashSystemCluster")
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

			By("By querying the created FlashSystemCluster")
			fscLoopUpKey := types.NamespacedName{
				Name:      FlashSystemName,
				Namespace: namespace,
			}
			createdFsc := &odfv1alpha1.FlashSystemCluster{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, fscLoopUpKey, createdFsc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By creating the ConfigMap")
			selectLabels := util.GetLabels()
			createdCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.PoolConfigmapName,
					Namespace: namespace,
					Labels:    selectLabels,
				},
			}
			createdCm.Data = make(map[string]string)
			value := util.FlashSystemClusterMapContent{
				ScPoolMap: make(map[string]string), Secret: secretName}
			val, _ := json.Marshal(value)
			createdCm.Data[FlashSystemName] = string(val)

			Eventually(func() bool {
				err := k8sClient.Create(ctx, createdCm)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("should create StorageClass successfully", func() {
			ctx := context.TODO()

			By("By creating a new StorageClass")
			sc := &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      storageClassName,
					Namespace: namespace,
				},
				Provisioner: util.CsiIBMBlockDriver,
				Parameters: map[string]string{
					"SpaceEfficiency": spaceEff,
					"pool":            poolName,
					"csi.storage.k8s.io/provisioner-secret-name":             secretName,
					"csi.storage.k8s.io/provisioner-secret-namespace":        namespace,
					"csi.storage.k8s.io/controller-publish-secret-name":      secretName,
					"csi.storage.k8s.io/controller-publish-secret-namespace": namespace,
					"csi.storage.k8s.io/controller-expand-secret-name":       secretName,
					"csi.storage.k8s.io/controller-expand-secret-namespace":  namespace,
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
				if err == nil {
					var sp util.FlashSystemClusterMapContent
					err = json.Unmarshal([]byte(createdCm.Data[FlashSystemName]), &sp)
					if err == nil {
						return sp.ScPoolMap[storageClassName] == poolName
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("should get FlashSystemCluster by StorageClass successfully", func() {
			ctx := context.TODO()

			By("creating a new secret with topology awareness")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      topologySecretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"config": []byte(topologySecretConfigData),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("By creating a new topology StorageClass")
			topologySc := &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      topologyStorageClassName,
					Namespace: namespace,
				},
				Provisioner: util.CsiIBMBlockDriver,
				Parameters: map[string]string{
					"by_management_id":                    byManagementIdData,
					"SpaceEfficiency":                     spaceEff,
					"pool":                                poolName,
					"csi.storage.k8s.io/secret-name":      topologySecretName,
					"csi.storage.k8s.io/secret-namespace": namespace,
					"csi.storage.k8s.io/fstype":           fsType,
					"volume_name_prefix":                  volPrefix,
				},
			}
			Expect(k8sClient.Create(ctx, topologySc)).Should(Succeed())

			By("By querying the created topology StorageClass")
			topologyScLookupKey := types.NamespacedName{
				Name:      topologyStorageClassName,
				Namespace: namespace,
			}
			createdTopologySc := &storagev1.StorageClass{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, topologyScLookupKey, createdTopologySc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By querying the ConfigMap to verify the topology StorageClass is added")
			cmLookupKey := types.NamespacedName{
				Name:      util.PoolConfigmapName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err == nil {
					var sp util.FlashSystemClusterMapContent
					err = json.Unmarshal([]byte(createdCm.Data[FlashSystemName]), &sp)
					if err == nil {
						return sp.ScPoolMap[topologyStorageClassName] == topologyPoolName
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			Expect(createdTopologySc.Name).To(Equal(topologyStorageClassName))
			Expect(createdTopologySc.Parameters["by_management_id"]).To(Equal(topologySc.Parameters["by_management_id"]))
			Expect(k8sClient.Delete(ctx, createdTopologySc)).Should(Succeed())

			By("By querying the ConfigMap to verify the topology StorageClass is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err == nil {
					var sp util.FlashSystemClusterMapContent
					err = json.Unmarshal([]byte(createdCm.Data[FlashSystemName]), &sp)
					if err == nil {
						return sp.ScPoolMap[topologyStorageClassName] == ""
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

		})

		It("should delete StorageClass successfully", func() {
			ctx := context.TODO()

			By("By deleting StorageClass")
			scLookupKey := types.NamespacedName{
				Name:      storageClassName,
				Namespace: "",
			}
			createdSc := &storagev1.StorageClass{}
			Expect(k8sClient.Get(ctx, scLookupKey, createdSc)).Should(Succeed())

			Expect(k8sClient.Delete(ctx, createdSc)).Should(Succeed())

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
			cmLookupKey := types.NamespacedName{
				Name:      util.PoolConfigmapName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}

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
