---
title: "Integrate Dapr with OSM"
description: "A simple demo showing to integrate Dapr with OSM"
type: docs
---

This document walks you through the steps of getting Dapr working with OSM on a kubernetes cluster.

1. Install Dapr on your cluster with mTLS disabled:

      1. Dapr has a quickstart repository to help users get familiar with dapr and its features. For this integration demo we will be leveraging the [hello-kubernetes](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes) quickstart. As we would like to integrate this Dapr example with OSM, there are a few modifications required and they are as follows:

         - The [hello-kubernetes](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes) demo installs Dapr with mtls enabled (by default), we would **not want mtls from Dapr and would like to leverage OSM for this**. Hence while [installing Dapr](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes#step-1---setup-dapr-on-your-kubernetes-cluster) on your cluster, make sure to disable mtls by passing the flag : `--enable-mtls=false`  during the installation
         - Futher [hello-kubernetes](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes) sets up everything in the default namespace, it is **strongly recommended** to set up the entire hello-kubernetes demo in a specific namespace (we will later join this namespace to OSM's mesh). For the purpose of this integration, we have the namespace as `dapr-demo`
         
           ```console
            $ kubectl create namespace dapr-demo
            namespace/dapr-demo created
           ```

         - The [redis state store](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes#step-2---create-and-configure-a-state-store), [redis.yaml](https://github.com/dapr/quickstarts/blob/master/hello-kubernetes/deploy/redis.yaml), [node.yaml](https://github.com/dapr/quickstarts/blob/master/hello-kubernetes/deploy/node.yaml) and [python.yaml](https://github.com/dapr/quickstarts/blob/master/hello-kubernetes/deploy/python.yaml) need to be deployed in the `dapr-demo` namespace
          - Since the resources for this demo are set up in a custom namespace. We will need to add an rbac rule on the cluster for Dapr to have access to the secrets. Create the following role and role binding:

            ```bash
            kubectl apply -f - <<EOF
            ---
            apiVersion: rbac.authorization.k8s.io/v1
            kind: Role
            metadata:
              name: secret-reader
              namespace: dapr-test
            rules:
            - apiGroups: [""]
              resources: ["secrets"]
              verbs: ["get", "list"]
            ---

            kind: RoleBinding
            apiVersion: rbac.authorization.k8s.io/v1
            metadata:
              name: dapr-secret-reader
              namespace: dapr-test
            subjects:
            - kind: ServiceAccount
              name: default
            roleRef:
              kind: Role
              name: secret-reader
              apiGroup: rbac.authorization.k8s.io
            EOF
            ```
      2. Ensure the sample applications are running with Dapr as desired.

2. Install OSM:

    ```console
    $ osm install
    OSM installed successfully in namespace [osm-system] with mesh name [osm]
    ```

3. Enable permissive mode in OSM:

    ```console
    $ kubectl patch meshconfig osm-mesh-config -n osm-system -p '{"spec":{"traffic":{"enablePermissiveTrafficPolicyMode":true}}}'  --type=merge
    meshconfig.config.openservicemesh.io/osm-mesh-config patched
    ```
    This is necessary, so that the hello-kubernetes example works as is and no SMI policies are needed from the get go.

4. Exclude kubernetes API server IP from being intercepted by OSM's sidecar:

    1. Get the kubenetes API server cluster IP:
       ```console
       $ kubectl get svc -n default
       NAME         TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
       kubernetes   ClusterIP   10.0.0.1     <none>        443/TCP   1d
       ```
    2. Add this IP to the MeshConfig so that outbound traffic to it is excluded from interception by OSM's sidecar
       ```console
       $ kubectl patch meshconfig osm-mesh-config -n osm-system -p '{"spec":{"traffic":{"outboundIPRangeExclusionList":["10.0.0.1/32"]}}}'  --type=merge
       meshconfig.config.openservicemesh.io/osm-mesh-config patched
       ```

    It is necessary to exclude the kubernetes API server IP in OSM because Dapr leverges kubernetes secrets to access the redis state store in this demo. 
    
    *Note: If you have hardcoded the password in the Dapr component file, you may skip this step.*

5. Globally exclude ports from being intercepted by OSM's sidecar:

    1. Get the ports of Dapr's placement server (`dapr-placement-server`):
       ```console
       $ kubectl get svc -n dapr-system
       NAME                    TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)              AGE
       dapr-api                ClusterIP   10.0.172.245   <none>        80/TCP               2h
       dapr-dashboard          ClusterIP   10.0.80.141    <none>        8080/TCP             2h
       dapr-placement-server   ClusterIP   None           <none>        50005/TCP,8201/TCP   2h
       dapr-sentry             ClusterIP   10.0.87.36     <none>        80/TCP               2h
       dapr-sidecar-injector   ClusterIP   10.0.77.47     <none>        443/TCP              2h
       ```
    2. Get the ports of your redis state store from the [redis.yaml](https://github.com/dapr/quickstarts/blob/master/hello-kubernetes/deploy/redis.yaml), `6379`incase of this demo
    
    3. Add these ports to the MeshConfig so that outbound traffic to it is excluded from interception by OSM's sidecar

       ```console
       $ kubectl patch meshconfig osm-mesh-config -n osm-system -p '{"spec":{"traffic":{"outboundPortExclusionList":[50005,8201,6379]}}}'  --type=merge
       meshconfig.config.openservicemesh.io/osm-mesh-config patched
       ```

    It is necessary to globally exclude Dapr's placement server (`dapr-placement-server`) port from being intercepted by OSM's sidecar, as pods having Dapr on them would need to talk to Dapr's control plane. The redis state store also needs to be excluded so that Dapr's sidecar can route the traffic to redis, without being intercepted by OSM's sidecar.
    
    *Note: Globally excluding ports would result in all pods in OSM's mesh from not interceting any outbound traffic to the specified ports. If you wish to exclude the ports selectively only on pods that are running Darp, you may omit this step and follow the step mentioned below.*

6. Exclude ports from being intercepted by OSM's sidecar at pod level:

    1. Get the ports of Dapr's api and sentry (`dapr-sentry` and `dapr-api`):
       ```console
       $ kubectl get svc -n dapr-system
       NAME                    TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)              AGE
       dapr-api                ClusterIP   10.0.172.245   <none>        80/TCP               2h
       dapr-dashboard          ClusterIP   10.0.80.141    <none>        8080/TCP             2h
       dapr-placement-server   ClusterIP   None           <none>        50005/TCP,8201/TCP   2h
       dapr-sentry             ClusterIP   10.0.87.36     <none>        80/TCP               2h
       dapr-sidecar-injector   ClusterIP   10.0.77.47     <none>        443/TCP              2h
       ```
    
    2. Update the pod spec in both nodeapp ([node.yaml](https://github.com/dapr/quickstarts/blob/master/hello-kubernetes/deploy/node.yaml)) and pythonapp ([python.yaml](https://github.com/dapr/quickstarts/blob/master/hello-kubernetes/deploy/python.yaml)) to contain the following annotation: `openservicemesh.io/outbound-port-exclusion-list: "80"`

    Adding the annotation to the pod excludes Dapr's api (`dapr-api`) and sentry (`dapr-sentry`) port's from being intercepted by OSM's sidecar, as these pods would need to talk to Dapr's control plane.

7. Make OSM monitor the namespace that was used for the Dapr hello-kubernetes demo setup:

   ```console
   $ osm namespace add dapr-test
   Namespace [dapr-test] successfully added to mesh [osm]
   ```

8. Delete and re-deploy the Dapr hello-kubernetes pods:

   ```console
   $ kubectl delete -f ./deploy/node.yaml
   service "nodeapp" deleted
   deployment.apps "nodeapp" deleted
   ```

   ```console
   $ kubectl delete -f ./deploy/python.yaml
   deployment.apps "pythonapp" deleted
   ```

   ```console
   $ kubectl apply -f ./deploy/node.yaml
   service "nodeapp" created
   deployment.apps "nodeapp" created
   ```

   ```console
   $ kubectl apply -f ./deploy/python.yaml
   deployment.apps "pythonapp" created
   ```

   The pythonapp and nodeapp pods on restart will now have 3 containers each, indicating OSM's proxy sidecar has been successfully injected

   ```console
   $ kubectl get pods -n dapr-test
   NAME                         READY   STATUS    RESTARTS   AGE
   my-release-redis-master-0    1/1     Running   0          2h
   my-release-redis-slave-0     1/1     Running   0          2h
   my-release-redis-slave-1     1/1     Running   0          2h
   nodeapp-7ff6cfb879-9dl2l     3/3     Running   0          68s
   pythonapp-6bd9897fb7-wdmb5   3/3     Running   0          53s
   ```
  
9. Verify the Darp hello-kubernetes demo works as expected:

    1. Verify the nodeapp service using the steps documented [here](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes#step-3---deploy-the-nodejs-app-with-the-dapr-sidecar) 
    
    2. Verify the pythonapp documented [here](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes#step-6---observe-messages)

10. Applying SMI Traffic Policies:

    The demo so far illustrated permissive traffic policy mode in OSM whereby application connectivity within the mesh is automatically configured by `osm-controller`, therefore no SMI policy was required for the pythonapp to talk to the nodeapp.

    In order to see the same demo work with an SMI Traffic Policy, follow the steps outlined below: 

    1. Disable permissive mode: 

       ```console
       $ kubectl patch meshconfig osm-mesh-config -n osm-system -p '{"spec":{"traffic":{"enablePermissiveTrafficPolicyMode":false}}}'  --type=merge
       meshconfig.config.openservicemesh.io/osm-mesh-config patched
       ```
    
    2. Verify the pythonapp documented [here](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes#step-6---observe-messages) no longer causes the order ID to increment.

    3. Create a service account for nodeapp and pythonapp: 

       ```console
       $ kubectl create sa nodeapp -n dapr-test
       serviceaccount/nodeapp created
       ```

       ```console
       $ kubectl create sa pythonapp -n dapr-test
       serviceaccount/pythonapp created
       ```
  
    4. Update the role binding on the cluster to contain the newly created service accounts:

       ```bash
       kubectl apply -f - <<EOF
       ---
       kind: RoleBinding
       apiVersion: rbac.authorization.k8s.io/v1
       metadata:
         name: dapr-secret-reader
         namespace: dapr-test
       subjects:
       - kind: ServiceAccount
         name: default
       - kind: ServiceAccount
         name: nopdeapp
       - kind: ServiceAccount
         name: pythonapp
       roleRef:
         kind: Role
         name: secret-reader
         apiGroup: rbac.authorization.k8s.io
       EOF
       ```

    5. Apply the following SMI access control policies:

       Deploy SMI TrafficTarget
       ```bash
       kubectl apply -f - <<EOF
       ---
       kind: TrafficTarget
       apiVersion: access.smi-spec.io/v1alpha3
       metadata:
         name: pythodapp-traffic-target
         namespace: dapr-test
       spec:
         destination:
           kind: ServiceAccount
           name: nodeapp
           namespace: dapr-test
         rules:
         - kind: HTTPRouteGroup
           name: nodeapp-service-routes
           matches:
           - new-order
         sources:
         - kind: ServiceAccount
           name: pythonapp
           namespace: dapr-test
       EOF
       ```

       Deploy HTTPRouteGroup policy
       ```bash
       kubectl apply -f - <<EOF
       ---
       apiVersion: specs.smi-spec.io/v1alpha4
       kind: HTTPRouteGroup
       metadata:
         name: nodeapp-service-routes
         namespace: dapr-test
       spec:
         matches:
         - name: new-order
       EOF
       ```

    6. Update the pod spec in both nodeapp ([node.yaml](https://github.com/dapr/quickstarts/blob/master/hello-kubernetes/deploy/node.yaml)) and pythonapp ([python.yaml](https://github.com/dapr/quickstarts/blob/master/hello-kubernetes/deploy/python.yaml)) to contain their respective service accounts. Delete and re-deploy the Dapr hello-kubernetes pods

    7. Verify the Darp hello-kubernetes demo works as expected, shown [here](https://github.com/dapr/quickstarts/tree/master/hello-kubernetes#step-6---observe-messages)

11. Cleanup: 
    
    1. To clean up the Darp hello-kubernetes demo, clean the `dapr-test` namespace

       ```console
       $ kubectl delete ns dapr-test
       ```

    2. To uninstall Dapr, run

       ```console
       $ dapr uninstall --kubernetes
       ```

    3. To uninstall OSM, run

       ```console
       $ osm uninstall
       ```

