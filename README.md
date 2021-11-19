# kube-gke-crossaccount-example
testing x-account kubeapi access


#### set up workload identity

already enabled on the following clusters in `gxlb-asm-01` with ID namespace `gxlb-asm-01.svc.id.goog`:
- gke-us-central1
- gke-us-west2
- keda-pubsub


Create GSA:
gcloud iam service-accounts create x-cluster-gsa
gcloud projects add-iam-policy-binding gxlb-asm-01 \
    --member "serviceAccount:x-cluster-gsa@gxlb-asm-01.iam.gserviceaccount.com" \
    --role "roles/container.developer"
gcloud iam service-accounts add-iam-policy-binding x-cluster-gsa@gxlb-asm-01.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:gxlb-asm-01.svc.id.goog[default/x-cluster-ksa]"

kubectx gke-us-central1
kubectl apply -f k8s/service-account.yaml

#### build & run image
(from demo script folder)
pack build --builder gcr.io/buildpacks/builder:v1 --publish gcr.io/gxlb-asm-01/x-account:01
kubectl apply -f k8s/pod.yaml

#### script stuff to set up:
- add serving component
- add target cluster env var (used to target)
- respond to web requests by creating a namespace (or something else trivial)

