### Flexvolume 

Build the flexvolume in openstorage and create a binary. Copy the
compiled binary into the kubernetes plugin path.
```
$ cd libopenstorage/openstorage
$ make
$ cp ../../../bin/flexvolume /usr/libexec/kubernetes/kubelet-plugins/volume/exec/osd~flexvolume/flexvolume
```

Note: Start the k8s cluster before copying this binary, as volume
plugins are probed during Kubelet bootstrap and it calls the
flexvolume "init" function, but there is no receiver driver to execute
the init function.

### OSD as a DaemonSet in kuberenetes

Run the OSD daemon
```
$ kubectl create -f etc/specs/k8s/osd-daemon.yaml
```

Run a nginx pod which uses osd flexvolume

```
$ kubectl create -f etc/specs/k8s/nginx-btrfs.yaml
```

Note: The above does not work as expected. The expected volume is not
mounted as the /var/lib/kubelet/pods/<pod-id>/volumes/osd~flexvolume
hostPath is not shared amongs the pods.

### PWX as a DaemonSet in kubernetes

Run px-lite container as a daemonset
```
$ kubectl create -f etc/specs/k8s/px-lite-daemon.yaml
```

Run a nginx pod which uses osd flexvolume
```
$ kubectl create -f etc/specs/k8s/nginx-pxd.yaml
```
Note: The above does not work as expected. The expected volume is not
mounted as the /var/lib/kubelet/pods/<pod-id>/volumes/osd~flexvolume
hostPath is not shared amongs the pods.

### Running OSD/PWX container outside of kubernetes

Run px-lite container using docker

```
$ docker run --restart=always --name px-lite -d --net=host
--privileged=true \
-v /run/docker/plugins:/run/docker/plugins \
-v /var/lib/osd:/var/lib/osd \
-v /dev:/dev \
-v /etc/pwx:/etc/pwx \
-v /opt/pwx/bin:/export_bin \
-v /var/run/docker.sock:/var/run/docker.sock \
-v /var/cores:/var/cores \
-v /var/lib/kubelet:/var/lib/kubelet:shared \
--ipc=host \
portworx/px-lite:latest
```

Run nginx pod with a flexvolume

```
$ kubectl create -f nginx-pwx.yaml
```

Note: The above works as expected and the flexvolume specified volume
gets mounted on the nginx pod.
