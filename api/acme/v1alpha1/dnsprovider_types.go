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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DNSProviderSpec defines the desired state of DNSProvider
type DNSProviderSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Code  DNSProviderCode   `json:"code,omitempty"`
	Email string            `json:"email,omitempty"`
	Keys  map[string]string `json:"keys,omitempty"`
}

type DNSProviderCode string

const (
	Cloudflare   DNSProviderCode = "cloudflare"
	AliDNS       DNSProviderCode = "alidns"
	TencentCloud DNSProviderCode = "tencentcloud"
)

// DNSProviderStatus defines the observed state of DNSProvider
type DNSProviderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +genclient

// DNSProvider is the Schema for the dnsproviders API
type DNSProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSProviderSpec   `json:"spec,omitempty"`
	Status DNSProviderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DNSProviderList contains a list of DNSProvider
type DNSProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DNSProvider{}, &DNSProviderList{})
}
