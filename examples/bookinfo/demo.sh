#! /bin/bash -ex

case $1 in
    "help") ## prints help message
        echo "Run ./demo.sh watch-pods, ./demo.sh 1, etc. in the order shown below"
        grep '##' demo.sh | grep -v "_.*##"
        ;;
    "help-verbose") ## prints help message, including debug commands
        echo "Run ./demo.sh watch-pods, ./demo.sh 1, etc. in the order shown below"
        grep '##' demo.sh
            ;;
    "watch-pods") ## watch pods transition between states
        kubectl get pods --all-namespaces -w
            ;;
    "1") ## [open new terminal] initialize supergloo
        supergloo init
            ;;
    "2") ## deploy istio
        supergloo install istio --name istio --installation-namespace istio-system --mtls=true --auto-inject=true --prometheus
            ;;
    "3") ## label namespace for injection
        kubectl label namespace default istio-injection=enabled --overwrite
            ;;
    "4") ## deploy bookinfo sample applicaiton
        kubectl apply -f bookinfo.yaml
        ;;
    "4_a") ## delete bookinfo sample applicaiton
        kubectl delete -f bookinfo.yaml
            ;;
    "forward") ## port forward to http://localhost:9080
        kubectl port-forward -n default deployment/productpage-v1 9080
            ;;
    "5") ## [open a new terminal] send all traffic to the "weak" version of the app, reviews:v4 (verify: stars are always red)
        supergloo apply routingrule trafficshifting \
                  --name reviews-v4 \
                  --dest-upstreams supergloo-system.default-reviews-9080 \
                  --target-mesh supergloo-system.istio \
                  --destination supergloo-system.default-reviews-v4-9080:1
            ;;
    "6") ##"preview-failure" Observe the "worst case" failure, Expect to see: "Error fetching product reviews!"
        supergloo apply routingrule faultinjection abort http \
                  --target-mesh supergloo-system.istio \
                  -p 100 -s 500  --name fault-product-to-reviews \
                  --dest-upstreams supergloo-system.default-reviews-9080
        ;;
    "7") ##"preview-failure-cleanup" Remove the fault that causes the "worst case" failure, Expect stars to have returned
        kubectl delete routingrule -n supergloo-system fault-product-to-reviews
        ;;
    "8") ## Triger the weakness (make failure between reviews and ratings) and note that it produces the "worst-case" failure - Expect to see failure between product and reviews
        supergloo apply routingrule faultinjection abort http \
                  --target-mesh supergloo-system.istio \
                  -p 100 -s 500  --name fault-reviews-to-ratings \
                  --dest-upstreams supergloo-system.default-ratings-9080
        ;;
    "9") ## Deploy a more robust version of the reviews app and note that despite the fault, we avoid the "worst-case" failure - Expect the reviews to show up and smaller error: "Ratings service is currently unavailable"
        kubectl delete routingrule -n supergloo-system reviews-v4
        supergloo apply routingrule trafficshifting \
                  --name reviews-v3 \
                  --dest-upstreams supergloo-system.default-reviews-9080 \
                  --target-mesh supergloo-system.istio \
                  --destination supergloo-system.default-reviews-v3-9080:1
        ;;
    "10") ## cleanup fault - Expect the reviews (red stars) to return
        kubectl delete routingrule -n supergloo-system fault-reviews-to-ratings
        ;;
    "11") ## deploy glooshot (run locally for now)
        go run ../../cmd/glooshot/main.go
        ;;
    "12") ## deploy experiment
        kubectl apply -f fault-abort-ratings.yaml
        ;;
    "12_a") ## verify that routingrule was created
        kubectl get experiment abort-ratings -o yaml
        ;;
    "12_b") ## delete the experiment
        kubectl delete experiment abort-ratings
        ;;
    "cleanup-istio")
        # namespace (do in background to ignore not-exist error)
        kubectl delete ns istio-system &
        # cluster-scoped resources
        for i in `kubectl get customresourcedefinitions -o=jsonpath="{.items[*].metadata.name}"`; do echo $i |grep istio|xargs kubectl delete customresourcedefinition ; done
        for i in `kubectl get clusterrole -o=jsonpath="{.items[*].metadata.name}"`; do echo $i |grep istio|xargs kubectl delete clusterrole ; done
        for i in `kubectl get clusterrolebinding -o=jsonpath="{.items[*].metadata.name}"`; do echo $i |grep istio|xargs kubectl delete clusterrolebinding ; done
        # do in background to ignore not-exist error
        kubectl delete mutatingwebhookconfiguration istio-sidecar-injector &
        # namespace-scoped resources in namespaces other than istio-system
        for n in `kubectl get ns -o=jsonpath="{.items[*].metadata.name}"`; do
            echo $n;
            # delete each secret made by istio
            for i in `kubectl get secrets -n=$n -o=jsonpath="{.items[*].metadata.name}"`; do echo $i |grep istio|xargs kubectl delete secret -n=$n; done
        done
    ;;
    *)
    error
    exit 1
    ;;
esac
