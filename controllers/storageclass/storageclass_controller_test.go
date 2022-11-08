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
		SecondFlashSystemName    = "second-flashsystemcluster-sample"
		ThirdFlashSystemName     = "third-flashsystemcluster-sample"
		namespace                = "openshift-storage"
		secretName               = "fs-secret-sample"
		secondSecretName         = "second-fs-secret-sample"
		thirdSecretName          = "third-fs-secret-sample"
		topologySecretName       = "topology-secret"
		storageClassName         = "odf-flashsystemcluster"
		topologyStorageClassName = "topology-storageclass"
		poolName                 = "Pool0"
		topologyPoolName         = "demo-pool-1"
		fsType                   = "ext4"
		volPrefix                = "product"
		spaceEff                 = "thin"
		byManagementIdData       = "{\"flashsystemcluster-sample\":{\"pool\":\"demo-pool-1\",\"SpaceEfficiency\":\"dedup_compressed\",\"volume_name_prefix\":\"demo-prefix-1\"},\"second-flashsystemcluster-sample\":{\"volume_name_prefix\":\"demo-prefix-2\", \"io_group\": \"demo-iogrp\"}}"
		topologySecretConfigData = "{\"flashsystemcluster-sample\": {\"username\": \"ZnNkcml2ZXI=\",\"password\": \"ZnNkcml2ZXI=\",\"management_address\": \"OS4xMTAuNzAuOTY=\"},\"second-flashsystemcluster-sample\": {\"username\": \"ZnNkcml2ZXI=\",\"password\": \"ZnNkcml2ZXI=\",\"management_address\": \"OS4xMTAuMTEuMjM=\"}}"

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

		It("should get the FlashSystemClusters by StorageClass successfully", func() {
			ctx := context.TODO()

			By("By creating the new secrets for the additional FlashSystemClusters and the topology StorageClass")
			secretsToCreateMap := map[string]string{secondSecretName: "OS4xMTAuMTEuMjM=", thirdSecretName: "OS4xMTAuNzcuMTE=", topologySecretName: ""}
			for secretName, mgmtAddr := range secretsToCreateMap {
				sec := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"management_address": []byte(mgmtAddr),
						"password":           []byte("ZnNkcml2ZXI="),
						"username":           []byte("ZnNkcml2ZXI="),
					},
				}
				By("creating a new secret with topology awareness for the topology StorageClass")
				if secretName == topologySecretName {
					sec.Data = map[string][]byte{
						"config": []byte(topologySecretConfigData),
					}
				}

				Expect(k8sClient.Create(ctx, sec)).Should(Succeed())
				By("By querying the created Secret for the created FlashSystemCluster")
				secLookupKey := types.NamespacedName{
					Name:      secretName,
					Namespace: namespace,
				}
				createdSec := &corev1.Secret{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, secLookupKey, createdSec)
					return err == nil
				}, timeout, interval).Should(BeTrue())
			}

			By("By creating the additional FlashSystemClusters")
			fscToSecretMap := map[string]string{SecondFlashSystemName: secondSecretName, ThirdFlashSystemName: thirdSecretName}
			for fscName, secretName := range fscToSecretMap {
				instance := &odfv1alpha1.FlashSystemCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fscName,
						Namespace: namespace,
					},
					Spec: odfv1alpha1.FlashSystemClusterSpec{
						Name: fscName,
						Secret: corev1.SecretReference{
							Name:      secretName,
							Namespace: namespace,
						},
						InsecureSkipVerify: true,
						DefaultPool: &odfv1alpha1.StorageClassConfig{
							StorageClassName: topologyStorageClassName,
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
					Name:      fscName,
					Namespace: namespace,
				}
				createdFsc := &odfv1alpha1.FlashSystemCluster{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, fscLoopUpKey, createdFsc)
					return err == nil
				}, timeout, interval).Should(BeTrue())
			}

			cmLookupKey := types.NamespacedName{
				Name:      util.PoolConfigmapName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}
			err := k8sClient.Get(ctx, cmLookupKey, createdCm)
			if err == nil {
				for fscName, secretName := range fscToSecretMap {
					value := util.FlashSystemClusterMapContent{
						ScPoolMap: make(map[string]string), Secret: secretName}
					val, _ := json.Marshal(value)
					createdCm.Data[fscName] = string(val)
				}
				err := k8sClient.Update(ctx, createdCm)
				Expect(err).Should(BeNil())
			}

			By("By creating a new topology aware StorageClass")
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

			By("By querying CM and verifying: topology SCs are added, different pool names identified for FSCs, " +
				"SC with management address which defers from secret is rejected from CM.")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err == nil {
					for fsc, value := range createdCm.Data {
						fc := util.FlashSystemClusterMapContent{}
						err := json.Unmarshal([]byte(value), &fc)
						if err != nil {
							return false
						}
						if fsc == FlashSystemName && fc.ScPoolMap[topologyStorageClassName] != topologyPoolName {
							return false
						}
						if fsc == SecondFlashSystemName && fc.ScPoolMap[topologyStorageClassName] != poolName {
							return false
						}
						if fsc == ThirdFlashSystemName && fc.ScPoolMap[topologyStorageClassName] != "" {
							return false
						}
					}
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())

		})

		It("Should remove the deleted StorageClass from the ConfigMap while other StorageClass should remain", func() {
			ctx := context.TODO()

			By("By getting the ConfigMap and verifying the StorageClass is present")
			topologyScLookupKey := types.NamespacedName{
				Name:      topologyStorageClassName,
				Namespace: namespace,
			}
			createdTopologySc := &storagev1.StorageClass{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, topologyScLookupKey, createdTopologySc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			cmLookupKey := types.NamespacedName{
				Name:      util.PoolConfigmapName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}

			By("By deleting the topology StorageClass")
			Expect(k8sClient.Delete(ctx, createdTopologySc)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err == nil {
					for fsc, value := range createdCm.Data {
						fc := util.FlashSystemClusterMapContent{}
						err := json.Unmarshal([]byte(value), &fc)
						if err != nil {
							return false
						}
						if _, ok := fc.ScPoolMap[topologyStorageClassName]; !ok {
							if fsc == FlashSystemName && fc.ScPoolMap[storageClassName] == poolName {
								return true
							}
							return false
						}
						return false
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("should delete all StorageClasses successfully", func() {
			ctx := context.TODO()

			By("By deleting StorageClass")
			scLookupKey := types.NamespacedName{
				Name:      storageClassName,
				Namespace: namespace,
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

			By("By querying the ConfigMap to verify StorageClass is deleted")
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
