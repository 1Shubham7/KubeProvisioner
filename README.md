# KubeProvisioner

KubeProvisioner is a Kubernetes operator that lets you manage cloud infrastructure as native Kubernetes resources. Instead of switching between the AWS console, Terraform, or CLI tools, you declare the resources you need in YAML — and KubeProvisioner creates, monitors, and cleans them up automatically.

The operator is built to grow. It currently supports AWS, and is designed to add new resource types and new cloud providers (GCP, Azure) without changing how you interact with it.


## Setup

The Helm chart lives at `./dist/chart` and installs the CRDs, RBAC, and the controller in one step.

```sh
helm install kubeprovisioner --create-namespace -n kubeprovisioner \
  --values ./dist/chart/values.yaml \
  --set controllerManager.container.env.AWS_ACCESS_KEY_ID=<your-key-id> \
  --set controllerManager.container.env.AWS_SECRET_ACCESS_KEY=<your-secret> \
  ./dist/chart/
```

To upgrade an existing release:

```sh
helm upgrade kubeprovisioner -n kubeprovisioner \
  --values ./dist/chart/values.yaml \
  ./dist/chart/
```

To uninstall:

```sh
helm uninstall kubeprovisioner -n kubeprovisioner
```

---

## Usage

### EC2 Instance

```yaml
apiVersion: compute.cloud.com/v1
kind: Ec2Instance
metadata:
  name: my-web-server
spec:
  instanceType: t3.micro
  amiId: ami-0c55b159cbfafe1f0
  region: us-east-1
  keyPair: my-key
  subnet: subnet-abc123
  tags:
    env: dev
    team: platform
```

```sh
kubectl apply -f my-instance.yaml
kubectl get ec2instances -w
```

The `State`, `PublicIP`, and `InstanceID` columns populate once the instance is running. To delete:

```sh
kubectl delete ec2instance my-web-server
```

The operator terminates the EC2 instance on AWS and then removes the CR.

---

### S3 Bucket

```yaml
apiVersion: compute.cloud.com/v1
kind: S3Bucket
metadata:
  name: my-app-assets
spec:
  bucketName: my-app-assets-kube-provisioner
  region: us-east-1
  versioning: true
  tags:
    env: dev
    team: platform
```

```sh
kubectl apply -f my-bucket.yaml
kubectl get s3buckets -w
```

The `BucketName`, `State`, and `Endpoint` columns populate once the bucket is ready. To delete:

```sh
kubectl delete s3bucket my-app-assets
```

> **Note:** The bucket must be empty before deletion. The operator will retry until it is empty.

Sample manifests for both resources are in [`kubernetes/sample-manifests/`](./kubernetes/sample-manifests/).

---

## Local development (kind cluster)

**1. Start a kind cluster:**

```sh
kind create cluster --name kubeprovisioner
kubectl config use-context kind-kubeprovisioner
```

**2. Install via Helm:**

```sh
helm install kubeprovisioner --create-namespace -n kubeprovisioner \
  --values ./dist/chart/values.yaml \
  --set controllerManager.container.env.AWS_ACCESS_KEY_ID=<your-key-id> \
  --set controllerManager.container.env.AWS_SECRET_ACCESS_KEY=<your-secret> \
  ./dist/chart/
```

**Alternative — run the controller locally** (faster iteration without rebuilding the image):

```sh
make install   # installs only the CRDs
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
go run cmd/main.go
```

---

## Demo

The operator running in the helm.ReleaseNS namespace:

![Operator running](static/image.png)

Applying the sample EC2 instance manifest:

![Applying manifest](static/image-1.png)

Instance comes up running:

![Instance running](static/image-2.png)

AWS console confirms the instance is live:

![AWS console](static/image-3.png)

---

## Internal commands

**Build and push the image:**

```sh
make docker-build docker-push IMG=<registry>/kubeprovisioner:tag
```

**Install CRDs only:**

```sh
make install
```

**Deploy the controller with a custom image:**

```sh
make deploy IMG=<registry>/kubeprovisioner:tag
```

**Apply all sample CRs:**

```sh
kubectl apply -k config/samples/
```

**Delete all sample CRs:**

```sh
kubectl delete -k config/samples/
```

**Remove the CRDs:**

```sh
make uninstall
```

**Remove the controller:**

```sh
make undeploy
```

Run `make help` for the full list of targets.

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for details.
