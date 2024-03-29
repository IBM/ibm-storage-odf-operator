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

package util

import (
	"fmt"
	"time"

	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reasons for ibm storage odf events
const (
	// SuccessfulLaunchBlockCSIReason is added in an event when Block CSI is successfully launched.
	SuccessfulLaunchBlockCSIReason      = "SuccessfulLaunchBlockCSI"
	FailedLaunchBlockCSIReason          = "FailedLaunchBlockCSI"
	SuccessfulDetectBlockCSIReason      = "SuccessfulDetectBlockCSI"
	FailedLaunchBlockExporterReason     = "FailedLaunchBlockExporter"
	FailedEnsureSecretOwnershipReason   = "FailedEnsureSecretOwnership"
	FailedCreateServiceReason           = "FailedCreateService"
	FailedScPoolConfigMapReason         = "FailedEnsureScPoolConfigMap"
	FailedCreateServiceMonitorReason    = "FailedCreateServiceMonitor"
	FailedCreateStorageClassReason      = "FailedCreateStorageClass"
	DeletedDuplicatedStorageClassReason = "DeletedDuplicatedStorageClass"
	FailedCreatePromRuleReason          = "FailedCreatePromRule"
)

func InitK8sEvent(instance *odfv1alpha1.FlashSystemCluster, eventType, reason, message string) *corev1.Event {
	t := metav1.Time{Time: time.Now()}
	selectLabels := GetLabels()
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", instance.Name, t.UnixNano()),
			Namespace: instance.Namespace,
			Labels:    selectLabels,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:            instance.Kind,
			Namespace:       instance.Namespace,
			Name:            instance.Name,
			UID:             instance.UID,
			ResourceVersion: instance.ResourceVersion,
			APIVersion:      instance.APIVersion,
		},
		Reason:         reason,
		Message:        message,
		FirstTimestamp: t,
		LastTimestamp:  t,
		Count:          1,
		Type:           eventType,
	}

}
