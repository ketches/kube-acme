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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
	"k8s.io/klog/v2"
)

type DNSProvider struct {
	Name string
	Envs map[string]string
}

type User struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *User) GetEmail() string {
	return u.Email
}
func (u User) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func NewUser(email string) *User {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		klog.Errorf("Failed to generate private key for user[%s]: %v", email, err)
	}
	return &User{
		Email: email,
		key:   privateKey,
	}
}

type AcmeClient struct {
	user           *User
	legoClient     *lego.Client
	DomainProvider *DNSProvider
}

func NewClient(user *User, provider *DNSProvider) *AcmeClient {
	config := lego.NewConfig(user)
	lgoClient, err := lego.NewClient(config)
	if err != nil {
		klog.Errorf("Failed")
	}
	return &AcmeClient{
		user:           user,
		legoClient:     lgoClient,
		DomainProvider: provider,
	}
}

func (c *AcmeClient) ObtainCertificate(domain string) (*certificate.Resource, error) {
	c.setenvs()

	provider, err := dns.NewDNSChallengeProviderByName(c.DomainProvider.Name)
	if err != nil {
		return nil, err
	}

	err = c.legoClient.Challenge.SetDNS01Provider(provider, dns01.AddRecursiveNameservers([]string{"114.114.114.114:53"}), dns01.AddDNSTimeout(120*time.Second))
	if err != nil {
		return nil, err
	}

	reg, err := c.legoClient.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	c.user.Registration = reg

	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	cert, err := c.legoClient.Certificate.Obtain(request)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func (c *AcmeClient) setenvs() {
	if c != nil && c.DomainProvider != nil {
		for k, v := range c.DomainProvider.Envs {
			os.Setenv(k, v)
		}
	}
}
