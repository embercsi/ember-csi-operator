# ember-csi-operator
Operator to create, configure and manage ember-csi, a multivendor CSI for
Kubernetes and OpenShift.

## Quick Start
#### Installing the operator
The operator needs its own namespace, service account, security context, and a
few roles and bindings. For example, to install these on OpenShift >= 3.10:

    oc create -f deploy/00-pre.yaml -f deploy/01-scc.yaml -f deploy/02-operator.yaml

#### Deploy and configure a storage backend
You also need a storage backend, for example a lightweight Ceph pod for
development & testing:

    oc create -f deploy/examples/ceph-demo.yaml

> If your cluster runs multiple compute nodes you need to allow TCP traffic to
> the Ceph pod on port 6800-7300. Please have a look at
> [Ceph Network > Configuration > Reference](http://docs.ceph.com/docs/master/rados/configuration/network-config-ref/#mds-and-manager-ip-tables)
> and [OpenShift Cluster Administration > Documentation](https://docs.openshift.com/container-platform/3.11/admin_guide/iptables.html)
> for further reference.

To use the Ceph container, you need to provide the ceph.conf configuration file
and the keyring file as a secret. The following commands will extract these two
files once the pod is ready to use and create a secret:

    oc wait -n ceph-demo --timeout=300s --for=condition=Ready pod/ceph-demo-pod
    oc -n ceph-demo cp ceph-demo/ceph-demo-pod:/etc/ceph/ etc/ceph/
    echo -e "\n[client]\nrbd default features = 3\n" >> etc/ceph/ceph.conf
    tar cf system-files.tar etc/ceph/ceph.conf etc/ceph/ceph.client.admin.keyring
    oc create -n ember-csi secret generic sysfiles-secret --from-file=system-files.tar

Next, setup the storage backend using a custom resource file:

    oc create -f deploy/examples/drivers/ceph.yaml

Now verify that the pods are created and the storage class exists:

    oc get pods -n ember-csi
    NAME                                  READY     STATUS    RESTARTS   AGE
    ember-csi-operator-645585cdc8-m62mp   1/1       Running   0          2m
    my-ceph-controller-0                  3/3       Running   0          11s
    my-ceph-node-0-d6gg4                  2/2       Running   0          11s
    my-ceph-node-0-lfzx6                  2/2       Running   0          11s

	oc get storageclass -n ember-csi
    NAME                      PROVISIONER            AGE
    io.ember-csi.my-ceph-sc   io.ember-csi.my-ceph   15s


#### Using the backend for your pods
You're all set now! However, you likely want to test the deployment, so let's
create a pvc and pod for testing.

    oc create namespace demoapp
    oc create -n demoapp -f deploy/examples/pvc.yaml -f deploy/examples/app.yaml
    
Once the pvc and pod are up and running, it will look like this:

    oc describe -n demoapp pods my-csi-app | tail

    Type    Reason                  Age   From                        Message
    ----    ------                  ----  ----                        -------
    Normal  Scheduled               20s   default-scheduler           Successfully assigned demoapp/my-csi-app to node2.example.com
    Normal  SuccessfulAttachVolume  19s   attachdetach-controller     AttachVolume.Attach succeeded for volume "pvc-6c6b9dd986f411e9"
    Normal  Pulling                 7s    kubelet, node2.example.com  pulling image "busybox"
    Normal  Pulled                  2s    kubelet, node2.example.com  Successfully pulled image "busybox"
    Normal  Created                 2s    kubelet, node2.example.com  Created container
    Normal  Started                 2s    kubelet, node2.example.com  Started container

Looking inside the container you will notice that the provided volume has been
mounted:

    oc exec -n demoapp -it my-csi-app  -- df -h | grep -B 1 /data
    /var/lib/ember-csi/vols/e1e57b59-f290-408f-87fa-540509bbe8b5 975.9M      2.5M    906.2M   0% /data

#### Uninstall the deployment
Eventually you want to remove all the resources from your cluster. Just delete
the projects, security context and storage class:

    oc delete project demoapp
    oc delete -f deploy/examples/drivers/ceph.yaml -f deploy/examples/ceph-demo.yaml
    oc delete -f deploy/00-pre.yaml -f deploy/01-scc.yaml -f deploy/02-operator.yaml

## Next steps
Documentation is still a work in progress, but have a look into [docs/README.md](docs/README.md) for further infos.
