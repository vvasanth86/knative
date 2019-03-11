export CLUSTER_NAME=ott-dev-knative-cluster
export CLUSTER_ZONE=asia-south1
export PROJECT=ott-knative

gcloud projects create $PROJECT --set-as-default
gcloud config set core/project $PROJECT

gcloud services enable cloudapis.googleapis.com container.googleapis.com containerregistry.googleapis.com compute.googleapis.com

gcloud container clusters create $CLUSTER_NAME --zone=$CLUSTER_ZONE --cluster-version=latest --machine-type=n1-standard-2 --enable-autoscaling --min-nodes=1 --max-nodes=4 --enable-autorepair --scopes=service-control,service-management,compute-rw,storage-ro,cloud-platform,logging-write,monitoring-write,pubsub,datastore --num-nodes=1

kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud config get-value core/account)


kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio-crds.yaml && kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio.yaml

kubectl label namespace default istio-injection=enabled


kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/serving.yaml --filename https://github.com/knative/serving/releases/download/v0.4.0/monitoring.yaml