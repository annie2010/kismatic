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
