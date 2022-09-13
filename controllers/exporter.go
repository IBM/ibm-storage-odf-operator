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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	"github.com/IBM/ibm-storage-odf-operator/controllers/util"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sYAML "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// exporter related dependencies
	// FlashSystemClusterSpec.Name as resource name for multiple flashSystem clusters

	// ExporterClusterConfigMapMountPoint = "/cluster-configmap"
	ExporterClusterConfigMapVolumeName = "storageclass-pool"
	ServiceAccount                     = "ibm-storage-odf-operator"
	fsServiceName                      = "ibm-flashsystem-storage-service"
	fsDeploymentName                   = "ibm-flashsystem-storage-deployment"
	fsServiceMonitorName               = "ibm-flashsystem-storage-service-monitor"

	portMetrics    = "metrics"
	scrapeInterval = "1m"
	scrapeTimeout  = "20s"

	flashsystemPrometheusRuleFilepath = "/prometheus-rules/prometheus-flashsystem-rules.yaml"
	// ruleName                          = "prometheus-flashsystem-rules"

	// FlashsystemPrometheusRuleFileEnv is only for UT
	FlashsystemPrometheusRuleFileEnv = "TEST_FS_PROM_RULE_FILE"
)

// TODO: wrapper func for deployment name translation from cluster name
func getExporterDeploymentName() string {
	return fsDeploymentName
}

func getExporterMetricsServiceName() string {
	return fsServiceName
}

func getExporterMetricsServiceMonitorName() string {
	return fsServiceMonitorName
}

func isOwnerExist(ownerReferences []metav1.OwnerReference, newOwner metav1.OwnerReference) bool {
	for _, owner := range ownerReferences {
		if owner.UID == newOwner.UID {
			return true
		}
	}
	return false
}

func InitExporterMetricsService(instance *odfv1alpha1.FlashSystemCluster) *corev1.Service {
	serviceName := getExporterMetricsServiceName()
	labels := util.GetLabels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: instance.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:     portMetrics,
					Port:     int32(9100),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(9100),
						StrVal: "9100",
					},
				},
			},
			Selector: labels,
		},
	}
}

func updateExporterMetricsService(foundService *corev1.Service, expectedService *corev1.Service, newOwner metav1.OwnerReference) *corev1.Service {
	isChanged := false

	if foundService.Spec.Type != expectedService.Spec.Type {
		isChanged = true
	}

	if len(foundService.Spec.Ports) != 1 ||
		!reflect.DeepEqual(foundService.Spec.Ports[0], expectedService.Spec.Ports[0]) {

		isChanged = true
	}

	if !isOwnerExist(foundService.GetOwnerReferences(), newOwner) {
		foundService.SetOwnerReferences(append(foundService.GetOwnerReferences(), newOwner))
		isChanged = true
	}

	if isChanged {
		updatedService := foundService.DeepCopy()
		updatedService.Spec.Type = corev1.ServiceTypeClusterIP
		updatedService.Spec.Ports = []corev1.ServicePort{
			{
				Name:     portMetrics,
				Port:     int32(9100),
				Protocol: corev1.ProtocolTCP,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(9100),
					StrVal: "9100",
				},
			},
		}
		updatedService.Spec.Selector = util.GetLabels()
		return updatedService
	}

	return nil
}

func InitExporterDeployment(
	instance *odfv1alpha1.FlashSystemCluster,
	pullPolicy corev1.PullPolicy,
	image string) (*appsv1.Deployment, error) {

	var replicaOne int32 = 1

	deploymentName := getExporterDeploymentName()
	labels := util.GetLabels()

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: instance.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaOne,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            deploymentName,
							Image:           image,
							ImagePullPolicy: pullPolicy,
							Env: []corev1.EnvVar{
								{
									Name:  util.WatchNamespaceEnvVar,
									Value: instance.Namespace,
								}},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("0.5"),
									corev1.ResourceMemory: resource.MustParse("500Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
							Ports: []corev1.ContainerPort{
								//{
								//	Name:          "grpc",
								//	ContainerPort: 36111,
								//	Protocol:      corev1.ProtocolTCP,
								//},
								{
									Name:          portMetrics,
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: int32(9100),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: ExporterClusterConfigMapVolumeName, MountPath: util.FSCConfigmapMountPath},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(9100),
									}},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(9100),
									}},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
						},
					},
					ServiceAccountName: ServiceAccount,
					Volumes: []corev1.Volume{
						{
							Name: ExporterClusterConfigMapVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: util.PoolConfigmapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func updateExporterDeployment(found *appsv1.Deployment, expected *appsv1.Deployment, newOwner metav1.OwnerReference) *appsv1.Deployment {
	isExist := isOwnerExist(found.GetOwnerReferences(), newOwner)
	if !isExist {
		found.SetOwnerReferences(append(found.GetOwnerReferences(), newOwner))
	}
	if !reflect.DeepEqual(found.Spec, expected.Spec) || !isExist {
		updated := found.DeepCopy()
		updated.Spec = *expected.Spec.DeepCopy()
		return updated
	}

	return nil
}

func InitExporterMetricsServiceMonitor(instance *odfv1alpha1.FlashSystemCluster) *monitoringv1.ServiceMonitor {
	selectLabels := util.GetLabels()

	serviceMonitor := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fsServiceMonitorName,
			Namespace: instance.Namespace,
			Labels:    selectLabels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{instance.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: selectLabels,
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port:          portMetrics,
					Interval:      scrapeInterval,
					ScrapeTimeout: scrapeTimeout,
				},
			},
		},
	}
	return serviceMonitor
}

func updateExporterMetricsServiceMonitor(foundServiceMonitor *monitoringv1.ServiceMonitor,
	expectedServiceMonitor *monitoringv1.ServiceMonitor, newOwner metav1.OwnerReference) *monitoringv1.ServiceMonitor {

	isExist := isOwnerExist(foundServiceMonitor.GetOwnerReferences(), newOwner)
	if !isExist {
		foundServiceMonitor.SetOwnerReferences(append(foundServiceMonitor.GetOwnerReferences(), newOwner))
	}

	if reflect.DeepEqual(foundServiceMonitor.Spec, expectedServiceMonitor.Spec) && isExist {
		return nil
	}

	updatedServiceMonitor := foundServiceMonitor.DeepCopy()
	updatedServiceMonitor.Spec = *expectedServiceMonitor.Spec.DeepCopy()

	return updatedServiceMonitor
}

func getFlashsystemPrometheusRuleFilepath() string {
	file, found := os.LookupEnv(FlashsystemPrometheusRuleFileEnv)
	if found {
		return file
	}

	return flashsystemPrometheusRuleFilepath
}

func getPrometheusRules(instance *odfv1alpha1.FlashSystemCluster) (*monitoringv1.PrometheusRule, error) {
	ruleFile, err := ioutil.ReadFile(filepath.Clean(getFlashsystemPrometheusRuleFilepath()))
	if err != nil {
		return nil, fmt.Errorf("prometheusRules file could not be fetched. %v", err)
	}
	var promRule monitoringv1.PrometheusRule
	err = k8sYAML.NewYAMLOrJSONDecoder(bytes.NewBufferString(string(ruleFile)), 8192).Decode(&promRule)
	if err != nil {
		return nil, fmt.Errorf("prometheusRules could not be decoded. %v", err)
	}

	template := promRule.GetName()
	promRule.SetName(instance.Name)
	promRule.SetNamespace(instance.Namespace)

	labels := util.GetLabels()
	updateLabels := promRule.GetLabels()
	if updateLabels == nil {
		updateLabels = labels
	} else {
		for k, v := range labels {
			updateLabels[k] = v
		}
	}
	promRule.SetLabels(updateLabels)

	// update expression of rules
	for i, group := range promRule.Spec.Groups {
		for j, rule := range group.Rules {
			if rule.Expr.Type == intstr.String {
				promRule.Spec.Groups[i].Rules[j].Expr.StrVal = strings.ReplaceAll(rule.Expr.StrVal, template, instance.Name)
				promRule.Spec.Groups[i].Rules[j].Labels["managedBy"] = instance.Name
			}
		}
	}

	owner := []metav1.OwnerReference{
		{
			APIVersion: instance.APIVersion,
			Kind:       instance.Kind,
			Name:       instance.Name,
			UID:        instance.UID,
		},
	}

	promRule.ObjectMeta.SetOwnerReferences(owner)

	return &promRule, nil
}
