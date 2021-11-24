/*
Copyright © 2021 Google Inc.

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

package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"google.golang.org/api/container/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // register GCP auth provider
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// get project id from env vars
var fProjectId = flag.String("projectId", os.Getenv("PROJECT_ID"), "specify a project id to examine")

// get target cluster name from env vars
var fTargetCluster = flag.String("targetCluster", os.Getenv("TARGET_CLUSTER"), "specify a target cluster to write to")

// hack to hold the KubeConfig
var kc *api.Config

// default http handler for all other paths
func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello\n")
}

// create namespaces handler for HTTP requests; calls the createNamspace func
func namespaceHandler(w http.ResponseWriter, req *http.Request) {
	// check for namespace name in query params
	keys, ok := req.URL.Query()["name"]
	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'name' is missing")
		fmt.Fprintf(w, "Url Param 'name' is missing\n")
		return
	}
	key := keys[0]
	fmt.Println("Target cluster is", *fTargetCluster)
	fmt.Println("New namespace name is", key)
	//fmt.Println(kc.Clusters[*fTargetCluster])
	fmt.Fprintf(w, "Attempting to create namespace %s on cluster %s\n", key, *fTargetCluster)
	createNamespace(context.Background(), key)
}

// function to actually create the namespace
func createNamespace(ctx context.Context, nsName string) error {

	cfg, err := clientcmd.NewNonInteractiveClientConfig(*kc, *fTargetCluster, &clientcmd.ConfigOverrides{CurrentContext: *fTargetCluster}, nil).ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes configuration cluster=%s: %w", *fTargetCluster, err)
	}

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}

	ns, err := k8s.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		log.Printf("failed to create namespace %s on cluster %s\n", nsName, *fTargetCluster)
	} else {
		fmt.Println("Created namespace", ns.Name)
	}

	return nil
}

func main() {

	// grab kubeconfig and assign to global kc variable
	kubeConfig, err := getK8sClusterConfigs(context.Background(), *fProjectId)
	if err != nil {
		log.Fatal(err)
	}
	kc = kubeConfig
	//fmt.Println(kubeConfig.Clusters[*fTargetCluster])

	http.HandleFunc("/createnamespace/", namespaceHandler)
	http.HandleFunc("/", hello)
	http.ListenAndServe(":8080", nil)
}

func getK8sClusterConfigs(ctx context.Context, projectId string) (*api.Config, error) {
	svc, err := container.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("container.NewService: %w", err)
	}

	// Basic config structure
	ret := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   map[string]*api.Cluster{},  // Clusters is a map of referencable names to cluster configs
		AuthInfos:  map[string]*api.AuthInfo{}, // AuthInfos is a map of referencable names to user configs
		Contexts:   map[string]*api.Context{},  // Contexts is a map of referencable names to context configs
	}

	// Ask Google for a list of all kube clusters in the given project.
	resp, err := svc.Projects.Zones.Clusters.List(projectId, "-").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("clusters list project=%s: %w", projectId, err)
	}

	for _, f := range resp.Clusters {
		name := fmt.Sprintf("gke_%s_%s_%s", projectId, f.Zone, f.Name)
		cert, err := base64.StdEncoding.DecodeString(f.MasterAuth.ClusterCaCertificate)
		if err != nil {
			return nil, fmt.Errorf("invalid certificate cluster=%s cert=%s: %w", name, f.MasterAuth.ClusterCaCertificate, err)
		}
		// example: gke_my-project_us-central1-b_cluster-1 => https://XX.XX.XX.XX
		ret.Clusters[name] = &api.Cluster{
			CertificateAuthorityData: cert,
			Server:                   "https://" + f.Endpoint,
		}
		// Just reuse the context name as an auth name.
		ret.Contexts[name] = &api.Context{
			Cluster:  name,
			AuthInfo: name,
		}
		// GCP specific configation; use cloud platform scope.
		ret.AuthInfos[name] = &api.AuthInfo{
			AuthProvider: &api.AuthProviderConfig{
				Name: "gcp",
				Config: map[string]string{
					"scopes": "https://www.googleapis.com/auth/cloud-platform",
				},
			},
		}
	}

	return &ret, nil
}
