# kube-gke-crossaccount-example
goal: demonstrate cross-cluster KubeAPI access by creating namespaces in one cluster via a pod in another cluster

This is implemented by mapping the permissions of a [Google service account](https://cloud.google.com/iam/docs/service-accounts) (or `GSA`) to a Kubernetes service account (or `KSA`) via [GKE Workload Identity](https://cloud.google.com/blog/products/containers-kubernetes/introducing-workload-identity-better-authentication-for-your-gke-applications), which is then used by a pod to communicate with the KubeAPI. 


### Set up workload identity

These steps follow the [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) docs, and use values in my project, so to test this you'll have to modify the pod & service account YAML in the `k8s` folder to reference your own project, container image URI, service accounts, workload identity pool, target cluster (i.e. where the namespace gets created), etc.

Create GSA:

```
gcloud iam service-accounts create x-cluster-gsa
```

Bind the `roles/container.developer` role to the GSA. 
> WARNING: the `roles/container.developer` role isn't scoped to a specific cluster, so if the env vars aren't configured correctly for the pod spec, you could end up creating resources on the wrong cluster within the project
```
gcloud projects add-iam-policy-binding gxlb-asm-01 \
    --member "serviceAccount:x-cluster-gsa@gxlb-asm-01.iam.gserviceaccount.com" \
    --role "roles/container.developer"
```
```
gcloud iam service-accounts add-iam-policy-binding x-cluster-gsa@gxlb-asm-01.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:gxlb-asm-01.svc.id.goog[default/x-cluster-ksa]"
```
```
kubectx gke-us-central1
kubectl apply -f k8s/service-account.yaml
```
### Build & run image
(from demo script folder, using [Google build packs](https://github.com/GoogleCloudPlatform/buildpacks))
```
pack build --builder gcr.io/buildpacks/builder:v1 --publish gcr.io/gxlb-asm-01/x-account:04
```
```
kubectl apply -f k8s/pod.yaml
```

Test calling pod
```
kubectl port-forward x-cluster-test 8080:8080
```
```
curl 127.0.0.1:8080/createnamespace/?name=hello5
```