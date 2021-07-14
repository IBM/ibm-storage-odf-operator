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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FlashSystemCRFilePath = "/config/csi.ibm.com_v1_ibmblockcsi_cr.yaml"
	// FlashSystemCRFilePathEnvVar is only for UT
	FlashSystemCRFilePathEnvVar = "TEST_FS_CR_FILEPATH"
)

func InitDefaultStorageClass(instance *odfv1alpha1.FlashSystemCluster) *storagev1.StorageClass {
	selectLabels := util.GetLabels(instance.Name)
	secret := instance.Spec.Secret
	pool := instance.Spec.DefaultPool
	if pool == nil {
		return nil
	}

	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pool.StorageClassName,
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

// flashsystem CSI CR operation related

// refer to "github.com/IBM/ibm-block-csi-operator/pkg/apis/csi/v1"
// NAME            SHORTNAMES  APIVERSION    NAMESPACED      KIND
// ibmblockcsis      ibc     csi.ibm.com/v1   true         IBMBlockCSI
func getFlashSystemCSIOperatorResource() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "csi.ibm.com",
		Version: "v1",
		Kind:    "IBMBlockCSI",
	}
}

func getFlashSystemCRFilePath() string {
	file, found := os.LookupEnv(FlashSystemCRFilePathEnvVar)
	if found {
		return file
	}

	return FlashSystemCRFilePath
}

func LoadFlashSystemCRFromFile() (*unstructured.Unstructured, error) {

	crFile := getFlashSystemCRFilePath()
	fmt.Printf("cr file: %s", crFile)
	filebytes, err := ioutil.ReadFile(crFile)
	if err != nil {
		return nil, err
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(filebytes), 128)

	var rawObj runtime.RawExtension
	if err = decoder.Decode(&rawObj); err != nil {
		return nil, err
	}

	obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return nil, err
	}

	expectedGVK := getFlashSystemCSIOperatorResource()
	if gvk.Group != expectedGVK.Group ||
		gvk.Kind != expectedGVK.Kind ||
		gvk.Version != expectedGVK.Version {

		return nil, fmt.Errorf("decoded GVK: %v is unexpected: %v", gvk, expectedGVK)
	}

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	return unstructuredObj, nil
}

func CreateIBMBlockCSICR(dc dynamic.NamespaceableResourceInterface, namespace string) (*unstructured.Unstructured, error) {
	client := dc.Namespace(namespace)

	unstructuredObj, err := LoadFlashSystemCRFromFile()
	if err != nil {
		return unstructuredObj, err
	}

	unstructuredObj.SetNamespace(namespace)
	obj, err := client.Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})

	return obj, err
}

func GetIBMBlockCSIDynamicClient(config *rest.Config) (dynamic.NamespaceableResourceInterface, error) {
	gvk := getFlashSystemCSIOperatorResource()

	resoureIsFound := false
	res := metav1.APIResource{}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create discovery client, err:%v", err)
	}

	resList, err := discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return nil, fmt.Errorf("failed to get resource list, err:%v", err)
	}

	for _, resource := range resList.APIResources {
		//if a resource contains a "/" it's referencing a subresource. we don't support suberesource for now.
		if resource.Kind == gvk.Kind && !strings.Contains(resource.Name, "/") {
			res = resource
			res.Group = gvk.Group
			res.Version = gvk.Version
			resoureIsFound = true
			break
		}
	}

	if !resoureIsFound {
		return nil, fmt.Errorf("unable to find the resource:%v", gvk)
	}

	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unalble to create dynamic client:%v", err)
	}

	return dc.Resource(schema.GroupVersionResource{
		Group:    res.Group,
		Version:  res.Version,
		Resource: res.Name,
	}), nil
}

func GetAllNamespace(config *rest.Config) ([]string, error) {
	//namespaces := []string{}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	k8sclient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	namespaces := &corev1.NamespaceList{}
	err = k8sclient.List(context.TODO(), namespaces)
	if err != nil {
		return nil, err
	}

	namespaceList := []string{}
	for _, ns := range namespaces.Items {
		namespaceList = append(namespaceList, ns.Namespace)
	}

	return namespaceList, nil
}

func HasIBMBlockCSICRExisted(namespaces []string, dc dynamic.NamespaceableResourceInterface) (bool, error) {

	for _, ns := range namespaces {
		obj, err := dc.Namespace(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			} else {
				return false, err
			}
		}

		if obj != nil && len(obj.Items) > 0 {
			return true, nil
		}
	}

	return false, nil
}
