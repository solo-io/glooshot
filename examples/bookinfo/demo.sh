#! /bin/bash -ex

case $1 in
    "help")
        echo "various commands to help with the demo"
        ;;
    "1")
        supergloo init
        ;;
    "2")
        supergloo install istio --name istio --installation-namespace istio-system --mtls=true --auto-inject=true
        ;;
    "3")
        kubectl label namespace default istio-injection=enabled --overwrite
        ;;
    "4")
        kubectl apply -f bookinfo.yaml
        ;;
    "5") ## all traffic goes to reviews:v4 (stars are always red)
        supergloo apply routingrule trafficshifting \
                  --name reviews-v4 \
                  --dest-upstreams supergloo-system.default-reviews-9080 \
                  --target-mesh supergloo-system.istio \
                  --destination supergloo-system.default-reviews-v4-9080:1
        ;;
    "6") ##"preview-failure") # Expect to see: "Error fetching product reviews!"
        supergloo apply routingrule faultinjection abort http \
                  --target-mesh supergloo-system.istio \
                  -p 100 -s 500  --name fault-product-to-reviews \
                  --dest-upstreams supergloo-system.default-reviews-9080
    ;;
    "7") ##"preview-failure-cleanup") # Expect stars to have returned
        kubectl delete routingrule -n supergloo-system fault-product-to-reviews
    ;;
    "8") ## Triger the weakness (make failure between reviews and ratings) - Expect to see failure between product and reviews
        supergloo apply routingrule faultinjection abort http \
                  --target-mesh supergloo-system.istio \
                  -p 100 -s 500  --name fault-reviews-to-ratings \
                  --dest-upstreams supergloo-system.default-ratings-9080
    ;;
    "9") ## Deploy a more robust version of the reviews app - Expect the reviews to show up and smaller error: ""Ratings service is currently unavailable""
        kubectl delete routingrule -n supergloo-system reviews-v4
        supergloo apply routingrule trafficshifting \
                  --name reviews-v3 \
                  --dest-upstreams supergloo-system.default-reviews-9080 \
                  --target-mesh supergloo-system.istio \
                  --destination supergloo-system.default-reviews-v3-9080:1
    ;;
    "forward")
        kubectl port-forward -n default deployment/productpage-v1 9080
    ;;
    "cleanup-istio")
        # cluster-scoped resources
        for i in `kubectl get customresourcedefinitions -o=jsonpath="{.items[*].metadata.name}"`; do echo $i |grep istio|xargs kubectl delete customresourcedefinition ; done
        for i in `kubectl get clusterrole -o=jsonpath="{.items[*].metadata.name}"`; do echo $i |grep istio|xargs kubectl delete clusterrole ; done
        for i in `kubectl get clusterrolebinding -o=jsonpath="{.items[*].metadata.name}"`; do echo $i |grep istio|xargs kubectl delete clusterrolebinding ; done
        # namespace-scoped resources in namespaces other than istio-system
        for n in `kubectl get ns -o=jsonpath="{.items[*].metadata.name}"`; do
            echo $n;
            # delete each secret made by istio
            for i in `kubectl get secrets -n=$n -o=jsonpath="{.items[*].metadata.name}"`; do echo $i |grep istio|xargs kubectl delete secret -n=$n; done
        done
        # namespace (do last, since it will fail and exit the script if it has already been removed)
        kubectl delete ns istio-system
    ;;
    *)
    error
    exit 1
    ;;
esac
