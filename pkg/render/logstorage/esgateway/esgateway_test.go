// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package esgateway

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tigera/operator/pkg/apis"
	"github.com/tigera/operator/pkg/controller/certificatemanager"
	"github.com/tigera/operator/pkg/dns"
	"github.com/tigera/operator/pkg/tls/certificatemanagement"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "github.com/tigera/operator/api/v1"
	"github.com/tigera/operator/pkg/common"
	"github.com/tigera/operator/pkg/render"
	relasticsearch "github.com/tigera/operator/pkg/render/common/elasticsearch"
	"github.com/tigera/operator/pkg/render/common/podaffinity"
	rtest "github.com/tigera/operator/pkg/render/common/test"
	"github.com/tigera/operator/pkg/render/kubecontrollers"
)

type resourceTestObj struct {
	name string
	ns   string
	typ  runtime.Object
	f    func(resource runtime.Object)
}

var _ = Describe("ES Gateway rendering tests", func() {
	Context("ES Gateway deployment", func() {
		var installation *operatorv1.InstallationSpec
		var replicas int32
		clusterDomain := "cluster.local"

		BeforeEach(func() {
			installation = &operatorv1.InstallationSpec{
				ControlPlaneReplicas: &replicas,
				KubernetesProvider:   operatorv1.ProviderNone,
				Registry:             "testregistry.com/",
			}
			replicas = 2
		})

		It("should render an ES Gateway deployment and all supporting resources", func() {
			expectedResources := []resourceTestObj{
				{kubecontrollers.ElasticsearchKubeControllersUserSecret, common.OperatorNamespace(), &corev1.Secret{}, nil},
				{kubecontrollers.ElasticsearchKubeControllersVerificationUserSecret, render.ElasticsearchNamespace, &corev1.Secret{}, nil},
				{kubecontrollers.ElasticsearchKubeControllersSecureUserSecret, render.ElasticsearchNamespace, &corev1.Secret{}, nil},
				{ServiceName, render.ElasticsearchNamespace, &corev1.Service{}, nil},
				{RoleName, render.ElasticsearchNamespace, &rbacv1.Role{}, nil},
				{RoleName, render.ElasticsearchNamespace, &rbacv1.RoleBinding{}, nil},
				{ServiceAccountName, render.ElasticsearchNamespace, &corev1.ServiceAccount{}, nil},
				{DeploymentName, render.ElasticsearchNamespace, &appsv1.Deployment{}, nil},
				{relasticsearch.PublicCertSecret, common.OperatorNamespace(), &corev1.Secret{}, nil},
			}

			kp, bundle := getTLS(installation)
			component := EsGateway(&Config{
				Installation: installation,
				PullSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: "tigera-pull-secret"}},
				},
				ESGatewayKeyPair: kp,
				TrustedBundle:    bundle,
				KubeControllersUserSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersUserSecret, Namespace: common.OperatorNamespace()}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersVerificationUserSecret, Namespace: render.ElasticsearchNamespace}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersSecureUserSecret, Namespace: render.ElasticsearchNamespace}},
				},
				ClusterDomain:   clusterDomain,
				EsAdminUserName: "elastic",
			})

			createResources, _ := component.Objects()
			compareResources(createResources, expectedResources)
		})

		It("should render an ES Gateway deployment and all supporting resources when CertificateManagement is enabled", func() {
			secret, err := certificatemanagement.CreateSelfSignedSecret("", "", "", nil)
			Expect(err).NotTo(HaveOccurred())
			installation.CertificateManagement = &operatorv1.CertificateManagement{CACert: secret.Data[corev1.TLSCertKey]}
			expectedResources := []resourceTestObj{
				{kubecontrollers.ElasticsearchKubeControllersUserSecret, common.OperatorNamespace(), &corev1.Secret{}, nil},
				{kubecontrollers.ElasticsearchKubeControllersVerificationUserSecret, render.ElasticsearchNamespace, &corev1.Secret{}, nil},
				{kubecontrollers.ElasticsearchKubeControllersSecureUserSecret, render.ElasticsearchNamespace, &corev1.Secret{}, nil},
				{ServiceName, render.ElasticsearchNamespace, &corev1.Service{}, nil},
				{RoleName, render.ElasticsearchNamespace, &rbacv1.Role{}, nil},
				{RoleName, render.ElasticsearchNamespace, &rbacv1.RoleBinding{}, nil},
				{ServiceAccountName, render.ElasticsearchNamespace, &corev1.ServiceAccount{}, nil},
				{DeploymentName, render.ElasticsearchNamespace, &appsv1.Deployment{}, nil},
				{relasticsearch.PublicCertSecret, common.OperatorNamespace(), &corev1.Secret{}, nil},
			}
			kp, bundle := getTLS(installation)
			component := EsGateway(&Config{
				Installation: installation,
				PullSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: "tigera-pull-secret"}},
				},
				ESGatewayKeyPair: kp,
				TrustedBundle:    bundle,
				KubeControllersUserSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersUserSecret, Namespace: common.OperatorNamespace()}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersVerificationUserSecret, Namespace: render.ElasticsearchNamespace}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersSecureUserSecret, Namespace: render.ElasticsearchNamespace}},
				},
				ClusterDomain:   clusterDomain,
				EsAdminUserName: "elastic",
			})

			createResources, _ := component.Objects()
			compareResources(createResources, expectedResources)
		})

		It("should not render PodAffinity when ControlPlaneReplicas is 1", func() {
			var replicas int32 = 1
			installation.ControlPlaneReplicas = &replicas
			kp, bundle := getTLS(installation)
			component := EsGateway(&Config{
				Installation: installation,
				PullSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: "tigera-pull-secret"}},
				},
				ESGatewayKeyPair: kp,
				TrustedBundle:    bundle,
				KubeControllersUserSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersUserSecret, Namespace: common.OperatorNamespace()}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersVerificationUserSecret, Namespace: render.ElasticsearchNamespace}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersSecureUserSecret, Namespace: render.ElasticsearchNamespace}},
				},
				ClusterDomain:   clusterDomain,
				EsAdminUserName: "elastic",
			})

			resources, _ := component.Objects()
			deploy, ok := rtest.GetResource(resources, DeploymentName, render.ElasticsearchNamespace, "apps", "v1", "Deployment").(*appsv1.Deployment)
			Expect(ok).To(BeTrue())
			Expect(deploy.Spec.Template.Spec.Affinity).To(BeNil())
		})

		It("should render PodAffinity when ControlPlaneReplicas is greater than 1", func() {
			var replicas int32 = 2
			installation.ControlPlaneReplicas = &replicas
			kp, bundle := getTLS(installation)
			component := EsGateway(&Config{
				Installation: installation,
				PullSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: "tigera-pull-secret"}},
				},
				ESGatewayKeyPair: kp,
				TrustedBundle:    bundle,
				KubeControllersUserSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersUserSecret, Namespace: common.OperatorNamespace()}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersVerificationUserSecret, Namespace: render.ElasticsearchNamespace}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersSecureUserSecret, Namespace: render.ElasticsearchNamespace}},
				},
				ClusterDomain:   clusterDomain,
				EsAdminUserName: "elastic",
			})

			resources, _ := component.Objects()
			deploy, ok := rtest.GetResource(resources, DeploymentName, render.ElasticsearchNamespace, "apps", "v1", "Deployment").(*appsv1.Deployment)
			Expect(ok).To(BeTrue())
			Expect(deploy.Spec.Template.Spec.Affinity).NotTo(BeNil())
			Expect(deploy.Spec.Template.Spec.Affinity).To(Equal(podaffinity.NewPodAntiAffinity(DeploymentName, render.ElasticsearchNamespace)))
		})

		It("should apply controlPlaneNodeSelector correctly", func() {
			installation.ControlPlaneNodeSelector = map[string]string{"foo": "bar"}
			kp, bundle := getTLS(installation)
			component := EsGateway(&Config{
				Installation: installation,
				PullSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: "tigera-pull-secret"}},
				},
				ESGatewayKeyPair: kp,
				TrustedBundle:    bundle,
				KubeControllersUserSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersUserSecret, Namespace: common.OperatorNamespace()}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersVerificationUserSecret, Namespace: render.ElasticsearchNamespace}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersSecureUserSecret, Namespace: render.ElasticsearchNamespace}},
				},
				ClusterDomain:   clusterDomain,
				EsAdminUserName: "elastic",
			})

			resources, _ := component.Objects()
			d, ok := rtest.GetResource(resources, DeploymentName, render.ElasticsearchNamespace, "apps", "v1", "Deployment").(*appsv1.Deployment)
			Expect(ok).To(BeTrue())
			Expect(d.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
		})

		It("should apply controlPlaneTolerations correctly", func() {
			t := corev1.Toleration{
				Key:      "foo",
				Operator: corev1.TolerationOpEqual,
				Value:    "bar",
			}
			kp, bundle := getTLS(installation)
			installation.ControlPlaneTolerations = []corev1.Toleration{t}
			component := EsGateway(&Config{
				Installation: installation,
				PullSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: "tigera-pull-secret"}},
				},
				ESGatewayKeyPair: kp,
				TrustedBundle:    bundle,
				KubeControllersUserSecrets: []*corev1.Secret{
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersUserSecret, Namespace: common.OperatorNamespace()}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersVerificationUserSecret, Namespace: render.ElasticsearchNamespace}},
					{ObjectMeta: metav1.ObjectMeta{Name: kubecontrollers.ElasticsearchKubeControllersSecureUserSecret, Namespace: render.ElasticsearchNamespace}},
				},
				ClusterDomain:   clusterDomain,
				EsAdminUserName: "elastic",
			})

			resources, _ := component.Objects()
			d, ok := rtest.GetResource(resources, DeploymentName, render.ElasticsearchNamespace, "apps", "v1", "Deployment").(*appsv1.Deployment)
			Expect(ok).To(BeTrue())
			Expect(d.Spec.Template.Spec.Tolerations).To(ConsistOf(t))
		})
	})
})

func getTLS(installation *operatorv1.InstallationSpec) (certificatemanagement.KeyPairInterface, certificatemanagement.TrustedBundle) {
	scheme := runtime.NewScheme()
	Expect(apis.AddToScheme(scheme)).NotTo(HaveOccurred())
	cli := fake.NewClientBuilder().WithScheme(scheme).Build()
	certificateManager, err := certificatemanager.Create(cli, installation, dns.DefaultClusterDomain)
	Expect(err).NotTo(HaveOccurred())
	esDNSNames := dns.GetServiceDNSNames(render.TigeraElasticsearchGatewaySecret, render.ElasticsearchNamespace, dns.DefaultClusterDomain)
	gwKeyPair, err := certificateManager.GetOrCreateKeyPair(cli, render.TigeraElasticsearchGatewaySecret, render.ElasticsearchNamespace, esDNSNames)
	Expect(err).NotTo(HaveOccurred())
	trustedBundle := certificateManager.CreateTrustedBundle(gwKeyPair)
	Expect(cli.Create(context.Background(), certificateManager.KeyPair().Secret(common.OperatorNamespace()))).NotTo(HaveOccurred())
	return gwKeyPair, trustedBundle
}

func compareResources(resources []client.Object, expectedResources []resourceTestObj) {
	Expect(len(resources)).To(Equal(len(expectedResources)))
	for i, expectedResource := range expectedResources {
		resource := resources[i]
		actualName := resource.(metav1.ObjectMetaAccessor).GetObjectMeta().GetName()
		actualNS := resource.(metav1.ObjectMetaAccessor).GetObjectMeta().GetNamespace()

		Expect(actualName).To(Equal(expectedResource.name), fmt.Sprintf("Rendered resource has wrong name (position %d, name %s, namespace %s)", i, actualName, actualNS))
		Expect(actualNS).To(Equal(expectedResource.ns), fmt.Sprintf("Rendered resource has wrong namespace (position %d, name %s, namespace %s)", i, actualName, actualNS))
		Expect(resource).Should(BeAssignableToTypeOf(expectedResource.typ))
		if expectedResource.f != nil {
			expectedResource.f(resource)
		}
	}
}
