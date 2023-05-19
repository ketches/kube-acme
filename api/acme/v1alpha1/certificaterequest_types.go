/*
Copyright 2023 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CertificateRequestSpec defines the desired state of CertificateRequest
type CertificateRequestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Domain         string `json:"domain,omitempty"`
	SecretName     string `json:"secretName,omitempty"`
	DNSProviderRef string `json:"dns,omitempty"`
}

// CertificateRequestStatus defines the observed state of CertificateRequest
type CertificateRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Phase      CertificateRequestPhase       `json:"phase,omitempty"`
	Conditions []CertificateRequestCondition `json:"conditions,omitempty"`
}

type CertificateRequestPhase string

const (
	CertificateRequestPending   CertificateRequestPhase = "Pending"
	CertificateRequestApproving CertificateRequestPhase = "Approving"
	CertificateRequestApproved  CertificateRequestPhase = "Approved"
	CertificateRequestDenied    CertificateRequestPhase = "Denied"
	CertificateRequestFailed    CertificateRequestPhase = "Failed"
)

type CertificateRequestConditionType string

const (
	CertificateObtained          CertificateRequestConditionType = "CertificateObtained"
	CertificateSecretConstructed CertificateRequestConditionType = "CertificateSecretConstructed"
	CertificateSecretSynced      CertificateRequestConditionType = "CertificateSecretSynced"
)

type CertificateRequestCondition struct {
	Type    CertificateRequestConditionType `json:"type,omitempty"`
	Status  corev1.ConditionStatus          `json:"status,omitempty"`
	Reason  string                          `json:"reason,omitempty"`
	Message string                          `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +genclient

// CertificateRequest is the Schema for the certificaterequests API
type CertificateRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateRequestSpec   `json:"spec,omitempty"`
	Status CertificateRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CertificateRequestList contains a list of CertificateRequest
type CertificateRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CertificateRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CertificateRequest{}, &CertificateRequestList{})
}
