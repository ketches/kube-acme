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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acmev1alpha1 "github.com/ketches/kube-acme/api/acme/v1alpha1"
)

// CertificateReconciler reconciles a Certificate object
type CertificateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=acme.ketches.cn,resources=certificates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=acme.ketches.cn,resources=certificates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=acme.ketches.cn,resources=certificates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Certificate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *CertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling Certificate")

	cert := &acmev1alpha1.Certificate{}
	if err := r.Get(ctx, req.NamespacedName, cert); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	switch cert.Status.Phase {
	case "":
		cert.Status.Phase = acmev1alpha1.CertificatePending
		err := r.Status().Update(ctx, cert)
		return ctrl.Result{}, err
	case acmev1alpha1.CertificatePending:
		if err := r.Get(ctx, types.NamespacedName{Namespace: cert.Namespace, Name: cert.Spec.SecretName}, &corev1.Secret{}); err == nil {
			cert.Status.Conditions = append(cert.Status.Conditions, acmev1alpha1.CertificateCondition{
				Type:    acmev1alpha1.CertificateConditionType(acmev1alpha1.CertificateSecretSynced),
				Status:  corev1.ConditionFalse,
				Reason:  "SyncCertificateSecretFailed",
				Message: fmt.Sprintf("Secret %s already exists", types.NamespacedName{Namespace: cert.Namespace, Name: cert.Spec.SecretName}),
			})
			err := r.Status().Update(ctx, cert)
			return ctrl.Result{}, err
		}
		crt, key, err := r.obtainCertificate(ctx, cert)
		if err != nil {
			cert.Status.Phase = acmev1alpha1.CertificateFailed
			cert.Status.Conditions = append(cert.Status.Conditions, acmev1alpha1.CertificateCondition{
				Type:    acmev1alpha1.CertificateObtained,
				Status:  corev1.ConditionFalse,
				Reason:  "ObtainCertificateFailed",
				Message: err.Error(),
			})
			err := r.Status().Update(ctx, cert)
			return ctrl.Result{}, err
		}

		// Build kubernetes.io/tls secret
		r.Client.Get(ctx, req.NamespacedName, cert)

		secret, err := r.constructTlsSecret(cert, crt, key)
		if err != nil {
			cert.Status.Phase = acmev1alpha1.CertificateFailed
			cert.Status.Conditions = append(cert.Status.Conditions, acmev1alpha1.CertificateCondition{
				Type:    acmev1alpha1.CertificateSecretConstructed,
				Status:  corev1.ConditionFalse,
				Reason:  "ConstructCertificateSecretFailed",
				Message: err.Error(),
			})
			err := r.Status().Update(ctx, cert)
			return ctrl.Result{}, err
		}
		cert.Status.Phase = acmev1alpha1.CertificateIssuing
		cert.Status.Conditions = append(cert.Status.Conditions, acmev1alpha1.CertificateCondition{
			Type:   acmev1alpha1.CertificateSecretConstructed,
			Status: corev1.ConditionTrue,
			Reason: "ConstructCertificateSecretDone",
		})

		if err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
			if k8serrors.IsNotFound(err) {
				if err := r.Create(ctx, secret); err != nil {
					cert.Status.Phase = acmev1alpha1.CertificateFailed
					cert.Status.Conditions = append(cert.Status.Conditions, acmev1alpha1.CertificateCondition{
						Type:    acmev1alpha1.CertificateSecretSynced,
						Status:  corev1.ConditionFalse,
						Reason:  "SyncCertificateSecretFailed",
						Message: err.Error(),
					})
				}
			}
		}

		cert.Status.Phase = acmev1alpha1.CertificateIssued
		cert.Status.Conditions = append(cert.Status.Conditions, acmev1alpha1.CertificateCondition{
			Type:   acmev1alpha1.CertificateSecretSynced,
			Status: corev1.ConditionTrue,
			Reason: "SyncCertificateSecretDone",
		})
		cert.Status.TlsCrt = crt
		cert.Status.TlsKey = key
		err = r.Status().Update(ctx, cert)
		return ctrl.Result{}, err
	case acmev1alpha1.CertificateIssuing:
	case acmev1alpha1.CertificateIssued:
	case acmev1alpha1.CertificateDenied:
	default:
		cert.Status.Phase = acmev1alpha1.CertificateFailed
		r.Status().Update(ctx, cert)
	}

	// r.Update(ctx, cert)

	return ctrl.Result{}, nil
}

func (r CertificateReconciler) constructTlsSecret(cert *acmev1alpha1.Certificate, crt, key []byte) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cert.Spec.SecretName,
			Namespace: cert.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cert, cert.GroupVersionKind()),
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": crt,
			"tls.key": key,
		},
	}
	return secret, nil
}

func (r *CertificateReconciler) obtainCertificate(ctx context.Context, cert *acmev1alpha1.Certificate) (crt []byte, key []byte, err error) {
	issuer := &acmev1alpha1.Issuer{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      cert.Spec.Issuer,
		Namespace: cert.Namespace,
	}, issuer); err != nil {
		return nil, nil, err
	}

	acmeCli := NewClient(NewUser(issuer.Spec.Email), &DNSProvider{Name: string(issuer.Spec.Solver), Envs: issuer.Spec.Keys})

	resource, err := acmeCli.ObtainCertificate(cert.Spec.Domains)
	if err != nil {
		return nil, nil, err
	}

	crt = resource.Certificate
	key = resource.PrivateKey

	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&acmev1alpha1.Certificate{}).
		Complete(r)
}
