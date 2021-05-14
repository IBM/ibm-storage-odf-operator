package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition represents the state of the operator's
// reconciliation functionality.
// +k8s:deepcopy-gen=true
type Condition struct {
	Type ConditionType `json:"type" description:"type of condition."`

	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`

	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime" description:"last time we got an update on a given condition"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime" description:"last time the condition transit from one status to another"`
}

// ConditionType is the state of the operator's reconciliation functionality.
type ConditionType string

const (
	// ExporterCreated indicts exporter is launched by operator
	ExporterCreated ConditionType = "ExporterCreated"
	// ExporterReady is set from exporter and reason & message are provided if false condition
	ExporterReady ConditionType = "ExporterReady"
	// StorageClusterReady is set from exporter after query from flashsystem
	StorageClusterReady ConditionType = "StorageClusterReady"
	// ProvisionerCreated indicts the flashsystem CSI CR is created
	ProvisionerCreated ConditionType = "ProvisionerCreated"
	// ProvisionerReused indicts the existing flashsystem CSI CR is reused
	ProvisionerReused ConditionType = "ProvisionerReused"
	// ProvisionerReady reflects the status of flashsystem CSI CR
	ProvisionerReady ConditionType = "ProvisionerReady"
	// ConditionProgressing indicts the reconciling process is in progress
	ConditionProgressing ConditionType = "ReconcileProgressing"
	// ConditionReconcileComplete indicts the Reconcile function completes
	ConditionReconcileComplete ConditionType = "ReconcileComplete"
)

const (
	ReasonReconcileFailed    = "ReconcileFailed"
	ReasonReconcileInit      = "Init"
	ReasonReconcileCompleted = "ReconcileCompleted"
)
