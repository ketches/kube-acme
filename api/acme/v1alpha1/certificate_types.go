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

// CertificateSpec defines the desired state of Certificate
type CertificateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Domains    []string `json:"domains,omitempty"`
	SecretName string   `json:"secretName,omitempty"`
	Issuer     string   `json:"issuer,omitempty"`
}

// CertificateStatus defines the observed state of Certificate
type CertificateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	TlsCrt     []byte                 `json:"tlsCrt,omitempty"`
	TlsKey     []byte                 `json:"tlsKey,omitempty"`
	Phase      CertificatePhase       `json:"phase,omitempty"`
	Conditions []CertificateCondition `json:"conditions,omitempty"`
}

type CertificatePhase string

const (
	CertificatePending CertificatePhase = "Pending"
	CertificateIssuing CertificatePhase = "Issuing"
	CertificateIssued  CertificatePhase = "Issued"
	CertificateDenied  CertificatePhase = "Denied"
	CertificateFailed  CertificatePhase = "Failed"
)

type CertificateConditionType string

const (
	CertificateObtained          CertificateConditionType = "CertificateObtained"
	CertificateSecretConstructed CertificateConditionType = "CertificateSecretConstructed"
	CertificateSecretSynced      CertificateConditionType = "CertificateSecretSynced"
)

type CertificateCondition struct {
	Type    CertificateConditionType `json:"type,omitempty"`
	Status  corev1.ConditionStatus   `json:"status,omitempty"`
	Reason  string                   `json:"reason,omitempty"`
	Message string                   `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +genclient

// Certificate is the Schema for the certificates API
type Certificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateSpec   `json:"spec,omitempty"`
	Status CertificateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CertificateList contains a list of Certificate
type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Certificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Certificate{}, &CertificateList{})
}
