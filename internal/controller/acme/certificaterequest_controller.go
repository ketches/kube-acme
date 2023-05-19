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

// CertificateRequestReconciler reconciles a CertificateRequest object
type CertificateRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=acme.ketches.cn,resources=certificaterequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=acme.ketches.cn,resources=certificaterequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=acme.ketches.cn,resources=certificaterequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CertificateRequest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *CertificateRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling CertificateRequest")

	cr := &acmev1alpha1.CertificateRequest{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	switch cr.Status.Phase {
	case "":
		cr.Status.Phase = acmev1alpha1.CertificateRequestPending
		err := r.Status().Update(ctx, cr)
		return ctrl.Result{Requeue: true}, err
	case acmev1alpha1.CertificateRequestPending:
		cert, key, err := r.obtainCertificate(ctx, cr)
		if err != nil {
			cr.Status.Phase = acmev1alpha1.CertificateRequestFailed
			cr.Status.Conditions = append(cr.Status.Conditions, acmev1alpha1.CertificateRequestCondition{
				Type:    acmev1alpha1.CertificateObtained,
				Status:  corev1.ConditionFalse,
				Reason:  "ObtainCertificateFailed",
				Message: err.Error(),
			})
			err := r.Status().Update(ctx, cr)
			return ctrl.Result{}, err
		}

		// Build kubernetes.io/tls secret
		r.Client.Get(ctx, req.NamespacedName, cr)

		secret, err := r.constructTlsSecret(cr, cert, key)
		if err != nil {
			cr.Status.Phase = acmev1alpha1.CertificateRequestFailed
			cr.Status.Conditions = append(cr.Status.Conditions, acmev1alpha1.CertificateRequestCondition{
				Type:    acmev1alpha1.CertificateSecretConstructed,
				Status:  corev1.ConditionFalse,
				Reason:  "ConstructCertificateSecretFailed",
				Message: err.Error(),
			})
			err := r.Status().Update(ctx, cr)
			return ctrl.Result{}, err
		}
		cr.Status.Phase = acmev1alpha1.CertificateRequestApproving
		cr.Status.Conditions = append(cr.Status.Conditions, acmev1alpha1.CertificateRequestCondition{
			Type:   acmev1alpha1.CertificateSecretConstructed,
			Status: corev1.ConditionTrue,
			Reason: "ConstructCertificateSecretDone",
		})

		if err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
			if k8serrors.IsNotFound(err) {
				if err := r.Create(ctx, secret); err != nil {
					cr.Status.Phase = acmev1alpha1.CertificateRequestFailed
					cr.Status.Conditions = append(cr.Status.Conditions, acmev1alpha1.CertificateRequestCondition{
						Type:    acmev1alpha1.CertificateSecretSynced,
						Status:  corev1.ConditionFalse,
						Reason:  "SyncCertificateSecretFailed",
						Message: err.Error(),
					})
				}
			}
		}

		cr.Status.Phase = acmev1alpha1.CertificateRequestApproved
		cr.Status.Conditions = append(cr.Status.Conditions, acmev1alpha1.CertificateRequestCondition{
			Type:   acmev1alpha1.CertificateSecretSynced,
			Status: corev1.ConditionTrue,
			Reason: "SyncCertificateSecretDone",
		})
		err = r.Status().Update(ctx, cr)
		return ctrl.Result{}, err
	case acmev1alpha1.CertificateRequestApproving:
	case acmev1alpha1.CertificateRequestApproved:
	case acmev1alpha1.CertificateRequestDenied:
	default:
		cr.Status.Phase = acmev1alpha1.CertificateRequestFailed
		r.Status().Update(ctx, cr)
	}

	// r.Update(ctx, cr)

	return ctrl.Result{}, nil
}

func (r CertificateRequestReconciler) constructTlsSecret(cr *acmev1alpha1.CertificateRequest, cert, key []byte) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.SecretName,
			Namespace: cr.Namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": key,
		},
	}
	return secret, nil
}

// func (r *CertificateRequestReconciler) getDNSProvider(ctx context.Context, cr *acmev1alpha1.CertificateRequest) (*acmev1alpha1.DNSProvider, error) {
// 	provider := &acmev1alpha1.DNSProvider{}
// 	if err := r.Get(ctx, types.NamespacedName{
// 		Name:      cr.Spec.DNSProviderRef,
// 		Namespace: cr.Namespace,
// 	}, provider); err != nil {
// 		return nil, err
// 	}
// 	return provider, nil
// }

func (r *CertificateRequestReconciler) obtainCertificate(ctx context.Context, cr *acmev1alpha1.CertificateRequest) (cert []byte, key []byte, err error) {
	provider := &acmev1alpha1.DNSProvider{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      cr.Spec.DNSProviderRef,
		Namespace: cr.Namespace,
	}, provider); err != nil {
		return nil, nil, err
	}

	acmeCli := NewClient(NewUser(provider.Spec.Email), &DNSProvider{Name: string(provider.Spec.Code), Envs: provider.Spec.Keys})

	resource, err := acmeCli.ObtainCertificate(cr.Spec.Domain)
	if err != nil {
		return nil, nil, err
	}

	cert = resource.Certificate
	key = resource.PrivateKey

	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&acmev1alpha1.CertificateRequest{}).
		Complete(r)
}
