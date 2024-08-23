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

package acme

import (
	"context"

	acmev1alpha1 "github.com/ketches/kube-acme/api/acme/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1 "k8s.io/api/networking/v1"
)

// IngressReconciler reconciles a Ingress object
type IngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ingress object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling Ingress")

	ingress := &networkingv1.Ingress{}
	if err := r.Get(ctx, req.NamespacedName, ingress); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	issuerName, ok := ingress.Annotations["kube-acme.ketches.cn/issuer"]
	if !ok {
		return ctrl.Result{}, nil
	}

	if len(ingress.Spec.TLS) == 0 || len(ingress.Spec.TLS[0].Hosts) == 0 {
		return ctrl.Result{}, nil
	}

	domains := ingress.Spec.TLS[0].Hosts

	issuer := &acmev1alpha1.Issuer{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ingress.Namespace, Name: issuerName}, issuer); err != nil {
		klog.Errorf("Failed to get Issuer [%s]: %s", client.ObjectKeyFromObject(issuer), err)
		return ctrl.Result{}, err
	}

	desiredCert := &acmev1alpha1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ingress.Namespace,
			Name:      ingress.Spec.TLS[0].SecretName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ingress, ingress.GroupVersionKind()),
			},
		},
		Spec: acmev1alpha1.CertificateSpec{
			Domains:    domains,
			SecretName: ingress.Spec.TLS[0].SecretName,
			Issuer:     issuerName,
		},
	}

	currentCert := &acmev1alpha1.Certificate{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(desiredCert), currentCert); err != nil {
		if k8serrors.IsNotFound(err) {
			// Create certificate
			if err := r.Create(ctx, desiredCert); err != nil {
				klog.Errorf("Failed to create Certificate [%s]: %s", client.ObjectKeyFromObject(desiredCert), err)
				return ctrl.Result{}, err
			}
		} else {
			if !slices.Equal(desiredCert.Spec.Domains, currentCert.Spec.Domains) || desiredCert.Spec.Issuer != currentCert.Spec.Issuer {
				// Update certificate
				if err := r.Update(ctx, desiredCert); err != nil {
					klog.Errorf("Failed to update Certificate [%s]: %s", client.ObjectKeyFromObject(desiredCert), err)
					return ctrl.Result{}, err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(r)
}
