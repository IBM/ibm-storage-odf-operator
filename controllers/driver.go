package controllers

import (
	"strings"

	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetDefaultStorageClassName(clusterName string) string {
	return strings.ToLower(clusterName) + "-default"
}

func InitDefaultStorageClass(instance *odfv1alpha1.FlashSystemCluster) *storagev1.StorageClass {
	selectLabels := util.GetLabels(instance.Name)
	secret := instance.Spec.Secret
	pool := instance.Spec.DefaultPool
	if pool == nil {
		return nil
	}

	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetDefaultStorageClassName(instance.Name),
			Namespace: "",
			Labels:    selectLabels,
			// cluster scoped objects can't set OwnerReferences to namespaced
		},
		Provisioner: util.CsiIBMBlockDriver,
		Parameters: map[string]string{
			"SpaceEfficiency": pool.SpaceEfficiency,
			"pool":            pool.PoolName,
			"csi.storage.k8s.io/provisioner-secret-name":             secret.Name,
			"csi.storage.k8s.io/provisioner-secret-namespace":        secret.Namespace,
			"csi.storage.k8s.io/controller-publish-secret-name":      secret.Name,
			"csi.storage.k8s.io/controller-publish-secret-namespace": secret.Namespace,
			"csi.storage.k8s.io/controller-expand-secret-name":       secret.Name,
			"csi.storage.k8s.io/controller-expand-secret-namespace":  secret.Namespace,
			"csi.storage.k8s.io/fstype":                              pool.FsType,
			"volume_name_prefix":                                     pool.VolumeNamePrefix,
		},
	}
	return sc
}

func compareDefaultStorageClass(
	found *storagev1.StorageClass,
	expected *storagev1.StorageClass) bool {

	if (found.Provisioner == expected.Provisioner) &&
		(found.Parameters["SpaceEfficiency"] == expected.Parameters["SpaceEfficiency"]) &&
		(found.Parameters["pool"] == expected.Parameters["pool"]) &&
		(found.Parameters["volume_name_prefix"] == expected.Parameters["volume_name_prefix"]) {
		return true
	}

	return false
}
