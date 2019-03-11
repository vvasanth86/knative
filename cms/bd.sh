docker build -t vv1990/cms .
docker push vv1990/cms

kubectl delete -f cms.yaml
kubectl apply -f cms.yaml
