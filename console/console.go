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

package console

import (
	"context"
	"strings"

	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const MainBasePath = "/"
const CompatibilityBasePath = "/compatibility/"

func GetDeployment(namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ibm-odf-console",
			Namespace: namespace,
		},
	}
}

func GetService(port int, namespace string) *apiv1.Service {
	return &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ibm-odf-console-service",
			Namespace: namespace,
			Annotations: map[string]string{
				"service.alpha.openshift.io/serving-cert-secret-name": "ibm-odf-console-serving-cert",
			},
			Labels: map[string]string{
				"app": "ibm-odf-console",
			},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{Protocol: "TCP",
					TargetPort: intstr.IntOrString{IntVal: int32(port)},
					Port:       int32(port),
					Name:       "console-port",
				},
			},
			Selector: map[string]string{
				"app": "ibm-odf-console",
			},
			Type: "ClusterIP",
		},
	}
}

func GetConsolePluginCR(consolePort int, basePath string, serviceNamespace string) *consolev1alpha1.ConsolePlugin {
	return &consolev1alpha1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ibm-storage-odf-plugin",
		},
		Spec: consolev1alpha1.ConsolePluginSpec{
			DisplayName: "IBM Plugin",
			Service: consolev1alpha1.ConsolePluginService{
				Name:      "ibm-odf-console-service",
				Namespace: serviceNamespace,
				Port:      int32(consolePort),
				BasePath:  basePath,
			},
		},
	}
}

/* +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
+kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=*
+kubebuilder:rbac:groups=operator.openshift.io,resources=consoles,verbs=* */

// RemoveConsole ensure plugin is cleaned when uninstall operator
func RemoveConsole(client client.Client, namespace string) error {
	consolePlugin := consolev1alpha1.ConsolePlugin{}
	if err := client.Get(context.TODO(), types.NamespacedName{
		Name:      "ibm-storage-odf-plugin",
		Namespace: namespace,
	}, &consolePlugin); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	// Delete ibm ConsolePlugin
	if err := client.Delete(context.TODO(), &consolePlugin); err != nil {
		return err
	}
	return nil
}
func GetBasePath(clusterVersion string) string {
	if strings.Contains(clusterVersion, "4.12") {
		return CompatibilityBasePath
	}

	return MainBasePath
}

func EnableIBMConsoleByDefault(client client.Client) error {
	var err error
	ibmConsoleName := "ibm-storage-odf-plugin"
	consoleCluster := operatorv1.Console{}
	if err = client.Get(context.TODO(), types.NamespacedName{
		Name: "cluster",
	}, &consoleCluster); err != nil {
		return err
	}
	consolePlugins := consoleCluster.Spec.Plugins
	if !IsContain(consolePlugins, ibmConsoleName) {
		consolePlugins = append(consolePlugins, ibmConsoleName)
		consoleCluster.Spec.Plugins = consolePlugins
		err = client.Update(context.TODO(), &consoleCluster)
	}

	return err
}

func IsContain(items []string, item string) bool {
	if len(items) == 0 {
		return false
	}
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}
