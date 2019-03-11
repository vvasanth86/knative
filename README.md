# Installation

Follow these instructions to setup a GKE cluster: https://github.com/knative/docs/blob/master/install/Knative-with-GKE.md

SOLR: https://lucidworks.com/2019/02/07/running-solr-on-kubernetes-part-1/

# Setup


# Verify CAS

```
curl -v -H "Host: cas.default.example.com" http://35.200.183.60/authorize/12345?jwt=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjN9.PZLMJBT9OIVG2qgp9hQr685oVYFgRgWpcSPmNcw6y7M
```

# Visualize Topology (Weavescope)

Installation: https://www.weave.works/docs/scope/latest/installing/#k8s

```
kubectl port-forward -n weave "$(kubectl get -n weave pod --selector=weave-scope-component=app -o jsonpath='{.items..metadata.name}')" 4040

http://localhost:4040
```


# Monitoring

- Open K8s Proxy

```
kubectl proxy
```

## Logs (Kibana) 

```
http://localhost:8001/api/v1/namespaces/knative-monitoring/services/kibana-logging/proxy/app/kibana#/management/kibana/index
```

## Tracing (Zipkin)

```
http://localhost:8001/api/v1/namespaces/istio-system/services/zipkin:9411/proxy/zipkin/
```

## Graphana

```
kubectl port-forward --namespace knative-monitoring $(kubectl get pods --namespace knative-monitoring --selector=app=grafana --output=jsonpath="{.items..metadata.name}") 3000


http://localhost:3000
```

## Prometheus 

```
kubectl port-forward -n knative-monitoring $(kubectl get pods -n knative-monitoring --selector=app=prometheus --output=jsonpath="{.items[0].metadata.name}") 9090

http://localhost:9090
```

## Web Dashboard

https://kubernetes.io/docs/tasks/access-application-cluster/web-ui-dashboard/

# Demo Use Cases

## Start Serving

```
export CAS_DOMAIN=`kubectl get route cas --output jsonpath="{.status.domain}"`
export CMS_DOMAIN=`kubectl get route cms --output jsonpath="{.status.domain}"`
export IP=`kubectl get svc istio-ingressgateway --namespace istio-system --output jsonpath="{.status.loadBalancer.ingress[*].ip}"`
export JWT="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjN9.PZLMJBT9OIVG2qgp9hQr685oVYFgRgWpcSPmNcw6y7M"

while true; do sleep 3; curl -H "Host: $CAS_DOMAIN" -H "Authorization: BEARER $JWT" http://$IP/authorize/1000; echo -e '\n'; done
```

## Traffic Splitting

```
kubectl apply -f cas-service-blue-green.yaml 
```

## Config Updates

```
kubectl create configmap cas-app-config --from-file=cas-app-config.yaml -o yaml --dry-run | kubectl replace -f -
```

## Autoscale (Up & Down to Zero)

Generate Load:

```
hey -host $CAS_DOMAIN -c 10 -n 100 -H "Authorization: BEARER $JWT" http://$IP/authorize/1000
```

Monitor Pods scaling
```
http://localhost:3000/d/u_-9SIMiz/knative-serving-scaling-debugging
```

## Rolling Updates



# References

- http://12factor.net
- https://www.martinfowler.com/microservices/
- https://medium.com/knative/knative-v0-3-autoscaling-a-love-story-d6954279a67a
- https://developer.ibm.com/blogs/knctl-a-simpler-way-to-work-with-knative/

