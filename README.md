# yacr
Yet Another Container Runtime

## TODO
- [ ] Parse arguments
    - `yacr run <image> <command> <args>`
- [ ] Run specified program
- [ ] Give the container it's own hostname
    - [namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
    - [sethostname](https://www.man7.org/linux/man-pages/man2/sethostname.2.html)
    - [clone](https://www.man7.org/linux/man-pages/man2/clone.2.html)
- [ ] Isolate processes running inside the container from the host filesystem
    - [chroot](https://www.man7.org/linux/man-pages/man2/chroot.2.html)
- [ ] Isolate the processes within the container from the host processes
    - [mount](https://www.man7.org/linux/man-pages/man2/mount.2.html) the /proc virtual fs (unmount when terminating the container)
    - Isolate new mount from the rest of the host
        -  Create new mount namespace
        -  Stop sharing the mount namespace with the host ([unshare](https://www.man7.org/linux/man-pages/man2/unshare.2.html))
- [ ] Run the container rootless
    - Create new user namespace and set the mappings between the users on the host and container ([user_namespaces](https://man7.org/linux/man-pages/man7/user_namespaces.7.html))
- [ ] Limit the resources the container has available to it (memory, number of processes, CPU available).
    - [cgroups](https://man7.org/linux/man-pages/man7/cgroups.7.html)
- [ ] Pulling and running container images
    - Use the [Docker Registry HTTP API V2](https://distribution.github.io/distribution/spec/api/)
    - [Authenticate](https://distribution.github.io/distribution/spec/auth/token/)
    - [Fetch the manifest for the image you wish to download](https://distribution.github.io/distribution/spec/api/#pulling-an-image-manifest)
    - [Parse the manifest to identify the layers to be downloaded](https://distribution.github.io/distribution/spec/manifest-v2-2/)
    - [Fetch each layer listed in the manifest](https://distribution.github.io/distribution/spec/api/#pulling-a-layer)
    - Unzip the layers on top of each other to re-create the filesystem
    - Fetch the config data and store it ready
- [ ] Run the container image we’ve pulled down
    - chroot to the root of the image we’ve pulled
    - Parse the config data that we saved in the previous step, in particular set the environment variables and working directory.
