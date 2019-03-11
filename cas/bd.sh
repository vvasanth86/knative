docker build -t vv1990/cas .
docker push vv1990/cas

kubectl apply -f cas-service.yaml