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
	"time"

	odfv1alpha1 "github.com/IBM/ibm-storage-odf-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// refer to https://github.com/openshift/ocs-operator and https://github.com/openshift/custom-resource-status

var (
	// PhaseIgnored is used when a resource is ignored
	PhaseIgnored = "Ignored"
	// PhaseProgressing is used when SetProgressingCondition is called
	PhaseProgressing = "Progressing"
	// PhaseError is used when SetErrorCondition is called
	PhaseError = "Error"
	// PhaseReady is used when SetCompleteCondition is called
	PhaseReady = "Ready"
	// PhaseNotReady is used when waiting for system to be ready after reconcile is successful
	PhaseNotReady = "Not Ready"
)

// SetReconcileProgressingCondition sets the ProgressingCondition to True and other conditions to
// false or Unknown. Used when we are just starting to reconcile, and there are no existing
// conditions.
func SetReconcileProgressingCondition(conditions *[]odfv1alpha1.Condition, reason string, message string) {
	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ConditionReconcileComplete,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	})

	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ConditionProgressing,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})

	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ProvisionerReady,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	})
}

// SetReconcileErrorCondition sets the ConditionReconcileComplete to False in
// case of any errors during the reconciliation process.
func SetReconcileErrorCondition(conditions *[]odfv1alpha1.Condition, reason string, message string) {
	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ConditionReconcileComplete,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ConditionProgressing,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: "processing FlashSystem ODF resources",
	})
}

// SetReconcileProvisionerErrorCondition sets the ProvisionerReady to False in
// case CSI driver fails to start
func SetReconcileProvisionerErrorCondition(conditions *[]odfv1alpha1.Condition, reason string, message string) {
	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ConditionReconcileComplete,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})

	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ProvisionerReady,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

// SetReconcileCompleteCondition sets the ConditionReconcileComplete to True and other Conditions
// to indicate that the reconciliation process has completed successfully.
func SetReconcileCompleteCondition(conditions *[]odfv1alpha1.Condition, reason string, message string) {
	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ConditionReconcileComplete,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})

	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ConditionProgressing,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})

	SetStatusCondition(conditions, odfv1alpha1.Condition{
		Type:    odfv1alpha1.ProvisionerReady,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
}

// SetStatusCondition sets the corresponding condition in conditions to newCondition.
func SetStatusCondition(conditions *[]odfv1alpha1.Condition, newCondition odfv1alpha1.Condition) {
	if conditions == nil {
		conditions = &[]odfv1alpha1.Condition{}
	}
	existingCondition := FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		newCondition.LastHeartbeatTime = metav1.NewTime(time.Now())
		*conditions = append(*conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
	existingCondition.LastHeartbeatTime = metav1.NewTime(time.Now())
}

// RemoveStatusCondition removes the corresponding conditionType from conditions.
func RemoveStatusCondition(conditions *[]odfv1alpha1.Condition, conditionType odfv1alpha1.ConditionType) {
	if conditions == nil {
		return
	}
	newConditions := []odfv1alpha1.Condition{}
	for _, condition := range *conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}

	*conditions = newConditions
}

// FindStatusCondition finds the conditionType in conditions.
func FindStatusCondition(conditions []odfv1alpha1.Condition, conditionType odfv1alpha1.ConditionType) *odfv1alpha1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}

	return nil
}

// IsStatusConditionTrue returns true when the conditionType is present and set to `corev1.ConditionTrue`
func IsStatusConditionTrue(conditions []odfv1alpha1.Condition, conditionType odfv1alpha1.ConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionTrue)
}

// IsStatusConditionFalse returns true when the conditionType is present and set to `corev1.ConditionFalse`
func IsStatusConditionFalse(conditions []odfv1alpha1.Condition, conditionType odfv1alpha1.ConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionFalse)
}

// IsStatusConditionPresentAndEqual returns true when conditionType is present and equal to status.
func IsStatusConditionPresentAndEqual(conditions []odfv1alpha1.Condition, conditionType odfv1alpha1.ConditionType, status corev1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}
