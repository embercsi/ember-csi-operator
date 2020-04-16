# ember-csi-operator
Operator to create, configure and manage ember-csi, a multivendor CSI for
Kubernetes and OpenShift.

## Quick Start
### Installing the operator
#### Operatorhub installation
You can use the operatorhub catalog to deploy the Ember CSI operator if you're
using Openshift 4.  You'll find the Ember CSI operator in the "Storage"
section.

### Install the operator
> You can also use the latest development version of the operator. To do so,
> please add another catalog entry and use this instead of the default one:
>
> ```
> oc create -f deploy/examples/catalog.yaml
> sed -ie 's/community-operators/external-operators/g' deploy/examples/subscription.yaml
> ```

You can install the operator using the catalog within the webinterface or using
the command line like this:

    oc create -f deploy/examples/operatorgroup.yaml
    oc create -f deploy/examples/subscription.yaml

You need to wait until the operator has been installed, which might take a
few minutes. You can check if the pod is running using the following command:

    oc get -l name=ember-csi-operator pod
    NAME                                 READY   STATUS    RESTARTS   AGE
    ember-csi-operator-bb9777478-xz9c8   1/1     Running   0          67s


### Deploy and configure a storage backend
You also need a storage backend, for example a lightweight LVM/iscsi pod for
development & testing:

    oc create -f deploy/examples/lvmbackend.yaml

Next, setup the storage backend using a custom resource file:

    oc create -f deploy/examples/lvmdriver.yaml

Now verify that the pods are created and the storage class exists:

    oc get pods
    NAME                                 READY   STATUS    RESTARTS   AGE
    ember-csi-operator-67985dbc7-fb98c   1/1     Running   0          92s
    example-controller-0                 4/4     Running   0          83s
    example-node-0-tshkq                 2/2     Running   0          83s
    lvmiscsi                             1/1     Running   0          3m9s

    oc get storageclass
    NAME                      PROVISIONER            AGE
    example.ember-csi.io-sc   example.ember-csi.io   4m29s

### Using the backend for your pods
You're all set now! However, you likely want to test the deployment, so let's
create a pvc and pod for testing.

    oc create -f deploy/examples/pvc.yaml -f deploy/examples/app.yaml

Once the pvc and pod are up and running, it will look like this:

    oc describe pods my-csi-app | tail

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

    oc exec -it my-csi-app -- df -h /data
    /var/lib/ember-csi/vols/e1e57b59-f290-408f-87fa-540509bbe8b5 975.9M      2.5M    906.2M   0% /data

### Testing
There is also a script that uses [Code Ready Containers](https://code-ready.github.io/crc/)
and executes all of the above commands, making it easy to start testing:

    deploy/examples/crclvm.sh

## Next steps
Documentation is still a work in progress, but have a look into [docs/README.md](docs/README.md) for further infos.
