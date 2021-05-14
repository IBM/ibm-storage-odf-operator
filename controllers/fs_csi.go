package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

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

func LoadFlashSystemCRFromFile() (*unstructured.Unstructured, error) {

	filebytes, err := ioutil.ReadFile(FlashSystemCRFilePath)
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

func GetIBMBlockCSIDiscoveryClient(config *rest.Config) (dynamic.NamespaceableResourceInterface, error) {
	gvk := getFlashSystemCSIOperatorResource()

	resoureIsFound := false
	res := metav1.APIResource{}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create discovery client, err:%v", err)
	}

	resList, err := discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return nil, fmt.Errorf("failed to get resoruce list, err:%v", err)
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
		return nil, fmt.Errorf("unable to find the resourc:%v", gvk)
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

func IsIBMBlockCSIInstanceFound(namespaces []string, dc dynamic.NamespaceableResourceInterface) (bool, error) {

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
