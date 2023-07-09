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
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
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
		fakeSecretName           = "fake-secret-sample"
		topologySecretName       = "topology-secret"
		storageClassName         = "odf-flashsystemcluster"
		secondStorageClassName   = "second-odf-flashsystemcluster"
		topologyStorageClassName = "topology-storageclass"
		poolName                 = "Pool0"
		topologyPoolName         = "demo-pool-1"
		fsType                   = "ext4"
		volPrefix                = "product"
		spaceEff                 = "thin"
		byManagementIdData       = "{\"demo-management-id-1\":{\"pool\":\"demo-pool-1\",\"SpaceEfficiency\":\"dedup_compressed\",\"volume_name_prefix\":\"demo-prefix-1\"},\"demo-management-id-2\":{\"volume_name_prefix\":\"demo-prefix-2\", \"io_group\": \"demo-iogrp\"}}"
		topologySecretConfigData = "{\"demo-management-id-1\": {\"username\": \"ZnNkcml2ZXI=\",\"password\": \"ZnNkcml2ZXI=\",\"management_address\": \"OS4xMTAuNzAuOTY=\"},\"demo-management-id-2\": {\"username\": \"ZnNkcml2ZXI=\",\"password\": \"ZnNkcml2ZXI=\",\"management_address\": \"OS4xMTAuMTEuMjM=\"}}" // #nosec G101 - false positive

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
					Name:      util.FscCmName,
					Namespace: namespace,
					Labels:    selectLabels,
				},
			}
			createdCm.Data = make(map[string]string)
			value := util.FscConfigMapFscContent{
				ScPoolMap: make(map[string]string), Secret: secretName}
			val, _ := json.Marshal(value)
			createdCm.Data[FlashSystemName] = string(val)

			Eventually(func() bool {
				err := k8sClient.Create(ctx, createdCm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By creating the Pools ConfigMap")
			createdPoolsCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.PoolsCmName,
					Namespace: namespace,
					Labels:    selectLabels,
				},
			}
			createdPoolsCm.Data = make(map[string]string)
			poolsValue := util.PoolsConfigMapFscContent{PoolsMap: make(map[string]util.PoolsConfigMapPoolContent),
				SrcOG: "", DestOG: ""}

			poolsVal, _ := json.Marshal(poolsValue)
			createdPoolsCm.Data[FlashSystemName] = string(poolsVal)

			Eventually(func() bool {
				err := k8sClient.Create(ctx, createdPoolsCm)
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
				Name:      util.FscCmName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}
			err := k8sClient.Get(ctx, cmLookupKey, createdCm)
			if err == nil {
				for fscName, secretName := range fscToSecretMap {
					value := util.FscConfigMapFscContent{
						ScPoolMap: make(map[string]string), Secret: secretName}
					val, _ := json.Marshal(value)
					createdCm.Data[fscName] = string(val)
				}
				err := k8sClient.Update(ctx, createdCm)
				Expect(err).Should(BeNil())
			}

			poolsCmLookupKey := types.NamespacedName{
				Name:      util.PoolsCmName,
				Namespace: namespace,
			}
			createdPoolsCm := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, poolsCmLookupKey, createdPoolsCm)
			if err == nil {
				for fscName, _ := range fscToSecretMap {
					poolsValue := util.PoolsConfigMapFscContent{PoolsMap: make(map[string]util.PoolsConfigMapPoolContent),
						SrcOG: "", DestOG: ""}
					val, _ := json.Marshal(poolsValue)
					createdPoolsCm.Data[fscName] = string(val)
				}
				err := k8sClient.Update(ctx, createdPoolsCm)
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
						fc := util.FscConfigMapFscContent{}
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
				Name:      util.FscCmName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}

			By("By deleting the topology StorageClass")
			Expect(k8sClient.Delete(ctx, createdTopologySc)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err == nil {
					for fsc, value := range createdCm.Data {
						fc := util.FscConfigMapFscContent{}
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

		It("should test the getSecret function successfully", func() {
			ctx := context.TODO()

			By("By creating a new StorageClass with a non-existent secret")
			sc := &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secondStorageClassName,
					Namespace: namespace,
				},
				Provisioner: util.CsiIBMBlockDriver,
				Parameters: map[string]string{
					"SpaceEfficiency":                     spaceEff,
					"pool":                                poolName,
					"csi.storage.k8s.io/secret-name":      fakeSecretName,
					"csi.storage.k8s.io/secret-namespace": namespace,
					"csi.storage.k8s.io/fstype":           fsType,
					"volume_name_prefix":                  volPrefix,
				},
			}

			Expect(k8sClient.Create(ctx, sc)).Should(Succeed())

			By("By querying the created StorageClass")
			scLookupKey := types.NamespacedName{
				Name:      secondStorageClassName,
				Namespace: namespace,
			}
			createdSc := &storagev1.StorageClass{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, scLookupKey, createdSc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			_, err := util.GetStorageClassSecret(k8sClient, createdSc)
			Expect(err).To(HaveOccurred())
			Expect(k8sClient.Delete(ctx, createdSc)).Should(Succeed())

			By("By getting a StorageClass with a valid secret")
			scLookupKey = types.NamespacedName{
				Name:      storageClassName,
				Namespace: namespace,
			}
			validCreatedSc := &storagev1.StorageClass{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, scLookupKey, validCreatedSc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			secret, err := util.GetStorageClassSecret(k8sClient, validCreatedSc)
			Expect(err).ToNot(HaveOccurred())
			Expect(secret).ToNot(BeNil())

		})

		It("should test the extractPoolName function successfully", func() {
			ctx := context.TODO()
			watcher := &StorageClassWatcher{
				Log:    ctrl.Log.WithName("controllers").WithName("StorageClass"),
				Client: k8sClient,
				Scheme: scheme.Scheme,
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

			byMgmtId := make(map[string]interface{})
			err := json.Unmarshal([]byte(byManagementIdData), &byMgmtId)
			Expect(err).ToNot(HaveOccurred())

			extractedPoolName, err := watcher.extractPoolName(*createdTopologySc, byMgmtId, "demo-management-id-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(extractedPoolName).To(Equal(topologyPoolName))

			extractedPoolName, err = watcher.extractPoolName(*createdTopologySc, byMgmtId, "demo-management-id-2")
			Expect(err).ToNot(HaveOccurred())
			Expect(extractedPoolName).To(Equal(poolName))

			extractedPoolName, err = watcher.extractPoolName(*createdTopologySc, byMgmtId, "demo-management-id-3")
			Expect(err).To(HaveOccurred())
			Expect(extractedPoolName).To(Equal(""))

		})

		It("Should test the getFlashSystemClusterByStorageClass function successfully", func() {
			ctx := context.TODO()
			watcher := &StorageClassWatcher{
				Log:    ctrl.Log.WithName("controllers").WithName("StorageClass"),
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			topologyScLookupKey := types.NamespacedName{
				Name:      topologyStorageClassName,
				Namespace: namespace,
			}
			createdTopologySc := &storagev1.StorageClass{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, topologyScLookupKey, createdTopologySc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By testing with a topology aware StorageClass")
			fscToPoolsMap, err := watcher.getFlashSystemClusterByStorageClass(createdTopologySc, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(fscToPoolsMap)).To(Equal(2))

			fscToPoolsMap, err = watcher.getFlashSystemClusterByStorageClass(createdTopologySc, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(fscToPoolsMap)).To(Equal(0))

			scLookupKey := types.NamespacedName{
				Name:      storageClassName,
				Namespace: namespace,
			}
			createdSc := &storagev1.StorageClass{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, scLookupKey, createdSc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By testing with a regular StorageClass")
			fscToPoolsMap, err = watcher.getFlashSystemClusterByStorageClass(createdSc, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(fscToPoolsMap)).To(Equal(1))
			Expect(fscToPoolsMap[FlashSystemName]).To(Equal(poolName))

			fscToPoolsMap, err = watcher.getFlashSystemClusterByStorageClass(createdSc, true)
			Expect(err).To(HaveOccurred())
			Expect(len(fscToPoolsMap)).To(Equal(0))

		})

		It("Should test the addStorageClassFromConfigMap and removeStorageClassFromConfigMap functions successfully", func() {
			ctx := context.TODO()
			watcher := &StorageClassWatcher{
				Log:    ctrl.Log.WithName("controllers").WithName("StorageClass"),
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			cmLookupKey := types.NamespacedName{
				Name:      util.FscCmName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			poolsCmLookupKey := types.NamespacedName{
				Name:      util.PoolsCmName,
				Namespace: namespace,
			}
			createdPoolsCm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, poolsCmLookupKey, createdPoolsCm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// should fail with a non-existing FlashSystemCluster
			err := watcher.addStorageClassToConfigMaps(*createdCm, *createdPoolsCm, "fake-fsc", storageClassName, poolName)
			Expect(err).To(HaveOccurred())

			// should succeed with a valid FlashSystemCluster
			err = watcher.addStorageClassToConfigMaps(*createdCm, *createdPoolsCm, FlashSystemName, storageClassName, poolName)
			Expect(err).ToNot(HaveOccurred())

			// should not fail with a non-existing FlashSystemCluster - it returns nil and skips
			err = watcher.removeStorageClassFromConfigMaps(*createdCm, *createdPoolsCm, "fake-fsc", storageClassName)
			Expect(err).ToNot(HaveOccurred())

			// should succeed with a valid FlashSystemCluster
			err = watcher.removeStorageClassFromConfigMaps(*createdCm, *createdPoolsCm, FlashSystemName, storageClassName)
			Expect(err).ToNot(HaveOccurred())

			// should not fail with a non-existing StorageClass
			err = watcher.removeStorageClassFromConfigMaps(*createdCm, *createdPoolsCm, FlashSystemName, "fake-sc")
			Expect(err).ToNot(HaveOccurred())
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
				Name:      util.FscCmName,
				Namespace: namespace,
			}
			createdCm := &corev1.ConfigMap{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, cmLookupKey, createdCm)
				if err != nil {
					return false
				}

				var sp util.FscConfigMapFscContent
				err = json.Unmarshal([]byte(createdCm.Data[FlashSystemName]), &sp)
				Expect(err).ToNot(HaveOccurred())

				_, ok := sp.ScPoolMap[storageClassName]
				return !ok

			}, timeout, interval).Should(BeTrue())
		})
	})
})
