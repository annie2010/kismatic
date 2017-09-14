# Disconnected Installation

Certain organizations need to run Kubernetes clusters in air-gapped environments, and thus need to perform an installation that is completely disconnected from the internet. The process of performing an installation on nodes with no internet access is called a disconnected installation.

Being disconnected means that you will not use public repositories or registries to get binaries to your nodes. Instead, you will first sync a local package repository and container image registry with the packages and images required to operate a Kubernetes cluster.

## Prerequisites

* Local package repository that is accessible from all nodes. This repository must include the Kubernetes software packages and their transitive dependencies.

* Local docker registry that is accessible from all nodes. This registry must be seeded with the images required for the installation.

## Planning the installation
Before executing the validation or installation stages, you must let KET know that
it should perform a disconnected installation. The following plan file options
must be considered:

**disconnected_installation**: This field must be set to `true` when performing a
disconnected installation. When `true`, KET will:
1. Run preflight checks that are specific to disconnected installations (detailed below)
2. Use the local image registry for cluster components, instead of pulling them from
Docker Hub, GCR, or other public registries.

**disable_package_installation**: In most cases, KET is responsible for installing the required packages onto the cluster nodes. If, however, you want to control the installation of the packages, you can set this flag to `true` to prevent KET from installing the packages. More importantly, disabling package installation will enable a set of preflight checks that will ensure the packages have been installed on all nodes.

**package_repository_urls**: When set, KET will configure the listed URLs as package repositories on all nodes. This is useful when your nodes are not preconfigured with the local repositories that contain the Kubernetes packages.

## Validating prerequisites
The KET preflight checks will ensure that all the packages and images are
readily available to the nodes. During the validation stage, KET will:

* Verify that the nodes have access to the required packages. This is achieved using
the operating system's package manager. In CentOS and RHEL, for example, KET will use `yum`
to verify that the packages are available on the node.

* Verify that the container registry has the required images. This is achieved using the registry API.

## Performing the installation

Once the relevant options in the plan file have been set, and the local repository and local registry have been stood up, you are ready to perform the disconnected installation. 

At this point, you can run `kismatic install apply` to initiate the installation.

# Creating a local package repository

## RHEL / CentOS

### Install required utilities
We will use `reposync` to download the packages from upstream repositories, and `httpd` to expose our local repository over HTTP.

```
yum install yum-utils httpd createrepo
```

### Setup the upstream repositories

The kubernetes and docker RPM repositories must be configured on the node to pull the packages.

```
# Add docker repo
sudo bash -c 'cat <<EOF > /etc/yum.repos.d/docker.repo
[docker]
name=Docker
baseurl=https://yum.dockerproject.org/repo/main/centos/7/
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://yum.dockerproject.org/gpg
EOF'

# Add Kubernetes repo
sudo bash -c 'cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
        https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF'
```

### Create list of transitive dependencies 

With this script, we list all the transitive dependencies of the packages required for a KET installation. The final list is stored in `pkgs.txt`.

```
# Get the dependencies for the packages we require and write them to pkgs.txt
packages="docker-engine-1.12.6-1.el7.centos kubelet-1.7.4-0 kubectl-1.7.4-0"
for pkg in $packages
do
    # Add our pkg to the list
    echo $pkg >> pkgs.dup.txt
    # Get the dependency tree for the pkg
    repoquery --archlist=`uname -m`,noarch --tree-requires $pkg |
    # Remove the pkg from the output, as it's already in the list
    tail -n +2 | 
    # Do some output cleanup
    sed 's/[\\|]//g' | tr -s ' ' |
    # Select the package name fields
    cut -d ' ' -f 3 |
    # Sort and unique the package list
    sort | uniq >> pkgs.dup.txt
done
# De-duplicate the packages in the file
sort pkgs.dup.txt | uniq > pkgs.txt
rm -f pkgs.dup.txt
```

### Configure custom repos for syncing
To prevent downloading entire repositories, we need to configure repository sources that will only provide the packages we are interested in.

```
pkgs=$(cat pkgs.txt)
cat <<'EOF' > /etc/yum.repos.d/centos-base-limited.repo
[centos-base-limited]
name=CentOS-$releasever - Base
mirrorlist=http://mirrorlist.centos.org/?release=$releasever&arch=$basearch&repo=os&infra=$infra
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-7
EOF
echo "includepkgs=$(echo $pkgs)" >> /etc/yum.repos.d/centos-base-limited.repo

cat <<'EOF' > /etc/yum.repos.d/centos-updates-limited.repo
[centos-updates-limited]
name=CentOS-$releasever - Updates
mirrorlist=http://mirrorlist.centos.org/?release=$releasever&arch=$basearch&repo=updates&infra=$infra
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-7
EOF
echo "includepkgs=$(echo $pkgs)" >> /etc/yum.repos.d/centos-updates-limited.repo

cat <<EOF > /etc/yum.repos.d/docker-limited.repo
[docker-limited]
name=Docker
baseurl=https://yum.dockerproject.org/repo/main/centos/7/
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://yum.dockerproject.org/gpg
includepkgs=$(echo $pkgs)
EOF

cat <<EOF > /etc/yum.repos.d/kubernetes-limited.repo
[kubernetes-limited]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
        https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
includepkgs=$(echo $pkgs)
EOF
```

### Download the RPMs using reposync
Sync the desired packages to the local machine, and place them in `/var/www/html`.

```
reposync -p /var/www/html/ -r centos-base-limited -r kubernetes-limited -r docker-limited -r centos-updates-limited
```

### Create a repository
Now that we have the RPMs locally, we must generate the metadata files required for the repository.

```
for repo in `ls /var/www/html`
do 
    createrepo /var/www/html/$repo
done
```

### Start Apache server
We will use the Apache HTTP server for exposing the repository over HTTP.
```
systemctl enable httpd
systemctl start httpd
```

At this point, you should be able to access the three repositories that were created. For example, the repository that contains the kubernetes packages can be found at `http://localhost/kubernetes-limited/`
