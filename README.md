# KubeProvisioner

A Kubernetes operator that lets you manage AWS EC2 instances as native Kubernetes resources. Define your EC2 instances as custom resources (`Ec2Instance`) and the operator handles provisioning, lifecycle management, and cleanup automatically.

## Description

KubeProvisioner bridges Kubernetes and AWS by implementing a custom controller for the `Ec2Instance` CRD. When you apply an `Ec2Instance` manifest, the operator calls the AWS EC2 API to launch the instance, waits for it to reach the running state, and syncs the instance details (instance ID, public/private IP, DNS, state) back to the resource's status. When you delete the resource, the operator terminates the corresponding EC2 instance and waits for it to be fully terminated before removing the finalizer.

**What it manages:**
- EC2 instance creation (AMI, instance type, key pair, subnet, security groups, storage, tags)
- Instance status syncing (state, IPs, DNS names)
- EC2 instance termination on CR deletion (via finalizer)

## Demo

<!-- Screenshots will be added here -->

### Local development setup (kind cluster)

**1. Start a kind cluster and make sure it's the active context:**

```sh
kind create cluster --name kubeprovisioner
kubectl config use-context kind-kubeprovisioner
```

**2. Install the CRD into the cluster:**

```sh
make install
```

**3. Set your AWS credentials:**

```sh
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
```

**4. Run the controller:**

```sh
go run cmd/main.go
```

**5. Apply an `Ec2Instance` resource:**

```yaml
apiVersion: compute.cloud.com/v1
kind: Ec2Instance
metadata:
  name: my-instance
spec:
  instanceType: t3.micro
  amiId: ami-0abcdef1234567890
  region: us-east-1
  keyPair: my-key-pair
  subnet: subnet-0abc123
```

```sh
kubectl apply -f my-instance.yaml
```

**6. Watch the instance status update:**

```sh
kubectl get ec2instances -w
```

You'll see the `State`, `PublicIP`, and `InstanceID` columns populate once the instance is running.

**7. Delete the instance:**

```sh
kubectl delete ec2instance my-instance
```

The operator will terminate the EC2 instance on AWS and then remove the resource from Kubernetes.

## Getting Started

### Prerequisites
- go version v1.23.0+
- docker version 17.03+
- kubectl version v1.11.3+
- kind (for local development)
- Access to a Kubernetes v1.11.3+ cluster
- AWS credentials with EC2 permissions

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/ec2operator:tag
```

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/ec2operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/ec2operator:tag
```

**NOTE:** The makefile target mentioned above generates an `install.yaml`
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/ec2operator/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under `dist/chart`.

## Contributing

Contributions are welcome. Please open an issue or pull request on GitHub.

**NOTE:** Run `make help` for more information on all potential `make` targets.

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html).

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
