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
	_ "github.com/IBM/ibm-storage-odf-operator/api/v1"
	odfv1 "github.com/IBM/ibm-storage-odf-operator/api/v1"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

var _ = Describe("PersistentVolume Controller", func() {
	const (
		FlashSystemName          = "flashsystemcluster-sample"
		PersistentVolume         = "test-persistent-volume"
		topologyPV               = "second-topology-test-persistent-volume"
		namespace                = "openshift-storage"
		PersistentVolumeClaim    = "test-persistent-volume-claim"
		PvcForTopology           = "topology-test-persistent-volume-claim"
		storageClassName         = "odf-flashsystemcluster"
		topologyStorageClassName = "topology-storageclass"
		poolName                 = "Pool0"
		secretName               = "fs-secret-sample"
		topologySecretName       = "topology-secret"
		fsType                   = "ext4"
		volPrefix                = "product"
		spaceEff                 = "thin"
		volumeHandle             = "00000000000000000000000000000001"
		topologyVolumeHandle     = "SVC:demo-management-id-2:25;600507607182869980000000000054D3"
		byManagementIdData       = "{\"demo-management-id-1\":{\"pool\":\"demo-pool-1\",\"SpaceEfficiency\":\"dedup_compressed\",\"volume_name_prefix\":\"demo-prefix-1\"}," +
			"\"demo-management-id-2\":{\"volume_name_prefix\":\"demo-prefix-2\", \"io_group\": \"demo-iogrp\"}}"
		topologySecretConfigData = "{\"demo-management-id-1\": {\"username\": \"ZnNkcml2ZXI=\",\"password\": \"ZnNkcml2ZXI=\",\"management_address\": \"OS4xMTAuNzAuOTY=\"}," +
			"						   \"demo-management-id-2\": {\"username\": \"ZnNkcml2ZXI=\",\"password\": \"ZnNkcml2ZXI=\",\"management_address\": \"OS4xMTAuMTEuMjM=\"}}" // #nosec G101 - false positive

		timeout = time.Second * 20
		//duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a new PersistentVolume", func() {
		It("Should create: Namespace, Secret and StorageClass successfully", func() {
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

			By("By creating new secrets")
			secretsToCreateMap := map[string]string{secretName: "OS4xMTAuMTEuMjM=", topologySecretName: ""}
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
			}

			By("By creating new StorageClasses")
			scToCreate := map[string]string{storageClassName: secretName, topologyStorageClassName: topologySecretName}
			for scName, secret := range scToCreate {
				sc := &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name:      scName,
						Namespace: namespace,
					},
					Provisioner: util.CsiIBMBlockDriver,
					Parameters: map[string]string{
						"SpaceEfficiency":                     spaceEff,
						"pool":                                poolName,
						"csi.storage.k8s.io/secret-name":      secret,
						"csi.storage.k8s.io/secret-namespace": namespace,
						"csi.storage.k8s.io/fstype":           fsType,
						"volume_name_prefix":                  volPrefix,
					},
				}
				if scName == topologyStorageClassName {
					sc.Parameters["by_management_id"] = byManagementIdData
				}

				Expect(k8sClient.Create(ctx, sc)).Should(Succeed())

				By("By querying the created StorageClass")
				scLookupKey := types.NamespacedName{
					Name:      scName,
					Namespace: namespace,
				}
				createdSc := &storagev1.StorageClass{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, scLookupKey, createdSc)
					return err == nil
				}, timeout, interval).Should(BeTrue())
			}

			By("By creating a new FlashSystemCluster")
			instance := &odfv1.FlashSystemCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FlashSystemName,
					Namespace: namespace,
				},
				Spec: odfv1.FlashSystemClusterSpec{
					Name: FlashSystemName,
					Secret: corev1.SecretReference{
						Name:      secretName,
						Namespace: namespace,
					},
					InsecureSkipVerify: true,
					DefaultPool: &odfv1.StorageClassConfig{
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
			createdFsc := &odfv1.FlashSystemCluster{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, fscLoopUpKey, createdFsc)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("should create a PersistentVolumeClaim successfully", func() {
			var volumeMode = corev1.PersistentVolumeBlock
			By("By creating new PersistentVolumeClaims")
			ctx := context.TODO()

			pvcToCreateList := map[string]string{PersistentVolumeClaim: storageClassName, PvcForTopology: topologyStorageClassName}
			for pvcName, sc := range pvcToCreateList {
				scName := sc
				pvc := &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pvcName,
						Namespace: namespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
						VolumeMode:       &volumeMode,
						StorageClassName: &scName,
					},
				}
				Expect(k8sClient.Create(ctx, pvc)).Should(Succeed())

				By("By querying the created PersistentVolumeClaim")
				pvcLookupKey := types.NamespacedName{
					Name:      pvcName,
					Namespace: namespace,
				}
				createdPvc := &corev1.PersistentVolumeClaim{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, pvcLookupKey, createdPvc)
					return err == nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should create a PersistentVolume successfully", func() {
			By("By creating a new PersistentVolume")
			ctx := context.TODO()
			volumeMode := corev1.PersistentVolumeBlock

			pvToCreateList := map[string]map[string]string{PersistentVolume: {"sc": storageClassName, "pvc": PersistentVolumeClaim},
				topologyPV: {"sc": topologyStorageClassName, "pvc": PvcForTopology}}
			for pvName, pvObjects := range pvToCreateList {
				pv := &corev1.PersistentVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pvName,
						Namespace: namespace,
					},
					Spec: corev1.PersistentVolumeSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Capacity: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("512Gi"),
						},
						PersistentVolumeSource: corev1.PersistentVolumeSource{
							CSI: &corev1.CSIPersistentVolumeSource{
								Driver:       util.CsiIBMBlockDriver,
								VolumeHandle: volumeHandle,
								FSType:       fsType,
							},
						},
						ClaimRef: &corev1.ObjectReference{
							Kind:      util.PersistentVolumeClaimKind,
							Name:      pvObjects["pvc"],
							Namespace: namespace,
						},
						StorageClassName: pvObjects["sc"],
						VolumeMode:       &volumeMode,
					},
				}

				if pvName == topologyPV {
					pv.Spec.PersistentVolumeSource.CSI.VolumeHandle = topologyVolumeHandle
				}
				Expect(k8sClient.Create(ctx, pv)).Should(Succeed())

				By("By querying the created PersistentVolume")
				pvLookupKey := types.NamespacedName{
					Name:      pvName,
					Namespace: namespace,
				}
				createdPv := &corev1.PersistentVolume{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, pvLookupKey, createdPv)
					//fmt.Println(createdPv.Name)
					//fmt.Println(createdPv.Labels)
					//fmt.Println(createdPv.Spec.CSI.VolumeHandle)
					//fmt.Println("------------------")
					return err == nil && createdPv.Labels[util.OdfFsStorageSystemLabelKey] == FlashSystemName

				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should test the ensureStorageSystemLabel function successfully", func() {
			ctx := context.TODO()

			watcher := &PersistentVolumeWatcher{
				Log:    ctrl.Log.WithName("controllers").WithName("PersistentVolume"),
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Adding the StorageSystem label to the PV")
			pv := &corev1.PersistentVolume{}
			pvLookupKey := types.NamespacedName{
				Name:      PersistentVolume,
				Namespace: namespace,
			}
			Expect(k8sClient.Get(ctx, pvLookupKey, pv)).Should(Succeed())
			err := watcher.ensureStorageSystemLabel(pv)
			Expect(err).Should(Not(HaveOccurred()))
			Expect(pv.Labels[util.OdfFsStorageSystemLabelKey]).Should(Equal(FlashSystemName))

			By("Testing the addStorageSystemLabelToPV function with a topology StorageClass")
			pvForTopology := &corev1.PersistentVolume{}
			pvLookupKey = types.NamespacedName{
				Name:      topologyPV,
				Namespace: namespace,
			}

			Expect(k8sClient.Get(ctx, pvLookupKey, pvForTopology)).Should(Succeed())
			err = watcher.ensureStorageSystemLabel(pvForTopology)
			Expect(err).Should(Not(HaveOccurred()))
			Expect(pv.Labels[util.OdfFsStorageSystemLabelKey]).Should(Equal(FlashSystemName))
		})

		It("should test the getPVManagementAddress function successfully", func() {
			ctx := context.TODO()

			watcher := &PersistentVolumeWatcher{
				Log:    ctrl.Log.WithName("controllers").WithName("PersistentVolume"),
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Getting the Management Address from the PV")
			expectedMgmtAddress := "OS4xMTAuMTEuMjM="
			pv := &corev1.PersistentVolume{}
			pvLookupKey := types.NamespacedName{
				Name:      PersistentVolume,
				Namespace: namespace,
			}
			Expect(k8sClient.Get(ctx, pvLookupKey, pv)).Should(Succeed())
			extractedMgmtAddr, err := watcher.getPVManagementAddress(pv)
			Expect(err).Should(Not(HaveOccurred()))
			Expect(extractedMgmtAddr).Should(Equal(expectedMgmtAddress))

			By("Testing the getPVManagementAddress function with a topology StorageClass")
			pvForTopology := &corev1.PersistentVolume{}
			pvLookupKey = types.NamespacedName{
				Name:      topologyPV,
				Namespace: namespace,
			}
			Expect(k8sClient.Get(ctx, pvLookupKey, pvForTopology)).Should(Succeed())
			extractedMgmtAddr, err = watcher.getPVManagementAddress(pvForTopology)
			Expect(err).Should(Not(HaveOccurred()))
			Expect(extractedMgmtAddr).Should(Equal(expectedMgmtAddress))

			By("Testing the getPVManagementAddress function with a StorageClass and no secret")
			secretLookupKey := types.NamespacedName{
				Name:      secretName,
				Namespace: namespace,
			}
			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, secretLookupKey, secret)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())

			_, err = watcher.getPVManagementAddress(pv)
			Expect(err).Should(HaveOccurred())

			By("By recreating the Secret that was deleted in the previous test")
			sec := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"management_address": []byte("OS4xMTAuMTEuMjM="),
					"password":           []byte("ZnNkcml2ZXI="),
					"username":           []byte("ZnNkcml2ZXI="),
				},
			}
			Expect(k8sClient.Create(ctx, sec)).Should(Succeed())

			By("By querying the recreated Secret")
			secLookupKey := types.NamespacedName{
				Name:      secretName,
				Namespace: namespace,
			}
			createdSec := &corev1.Secret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secLookupKey, createdSec)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Testing the getPVManagementAddress function without a StorageClass")
			sc := &storagev1.StorageClass{}
			scLookupKey := types.NamespacedName{
				Name:      storageClassName,
				Namespace: namespace,
			}
			Expect(k8sClient.Get(ctx, scLookupKey, sc)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, sc)).Should(Succeed())

			_, err = watcher.getPVManagementAddress(pv)
			Expect(err).Should(HaveOccurred())

			By("By recreating the StorageClass that was deleted in the previous test")
			sc = &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      storageClassName,
					Namespace: namespace,
				},
				Provisioner: util.CsiIBMBlockDriver,
				Parameters: map[string]string{
					"SpaceEfficiency":                     spaceEff,
					"pool":                                poolName,
					"csi.storage.k8s.io/secret-name":      secretName,
					"csi.storage.k8s.io/secret-namespace": namespace,
					"csi.storage.k8s.io/fstype":           fsType,
					"volume_name_prefix":                  volPrefix,
				},
			}
			Expect(k8sClient.Create(ctx, sc)).Should(Succeed())

			By("By querying the recreated StorageClass")
			scLookupKey = types.NamespacedName{
				Name:      storageClassName,
				Namespace: namespace,
			}
			createdSc := &storagev1.StorageClass{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, scLookupKey, createdSc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

		})

	})
})
