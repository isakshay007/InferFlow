# Kubernetes Deployment

This document covers deploying InferFlow on AWS EKS with a public ALB endpoint.

## Manifests

- [router.yaml](../k8s/router.yaml)
- [redis.yaml](../k8s/redis.yaml)
- [vllm-worker.yaml](../k8s/vllm-worker.yaml)
- [ingress.yaml](../k8s/ingress.yaml)

## Prerequisites

- AWS account with EKS, ECR, IAM, and ELB permissions
- `aws` CLI configured (`aws configure sso` or access keys)
- `kubectl`, `helm` installed locally
- EKS cluster running (see `terraform/environments/aws/` for provisioning)
- Docker images pushed to ECR (see [Building Images](#building-images))

---

## Building Images

```bash
ECR="310429010516.dkr.ecr.us-east-1.amazonaws.com/inferflow"

# Authenticate
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 310429010516.dkr.ecr.us-east-1.amazonaws.com

# Router
docker build -f Dockerfile.router -t $ECR:router .
docker push $ECR:router

# vLLM adapter sidecar
docker build -f Dockerfile.vllm-adapter -t $ECR:vllm-adapter .
docker push $ECR:vllm-adapter

# llama.cpp inference server
docker build -f Dockerfile.llama-server -t $ECR:llama-server .
docker push $ECR:llama-server
```

> **Note:** `Dockerfile.llama-server` downloads the llama.cpp b8838 pre-built binary
> from GitHub at build time. First build takes ~30s.

---

## Deploy Core Services

```bash
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/vllm-worker.yaml   # StatefulSet: 3 llama.cpp workers
kubectl apply -f k8s/router.yaml
```

Wait for all pods to be ready:

```bash
kubectl get pods -l app=vllm-worker     # expect 3/3 pods at 2/2
kubectl get pods -l app=inferflow-router
```

---

## Expose via ALB (public URL)

This requires the **AWS Load Balancer Controller** installed once per cluster.

### 1. Register the OIDC provider

```bash
OIDC_ID=$(aws eks describe-cluster --name <your-cluster-name> --region us-east-1 \
  --query "cluster.identity.oidc.issuer" --output text | cut -d'/' -f5)

aws iam create-open-id-connect-provider \
  --url "https://oidc.eks.us-east-1.amazonaws.com/id/${OIDC_ID}" \
  --client-id-list sts.amazonaws.com \
  --thumbprint-list "9e99a48a9960b14926bb7f3b02e22da2b0ab7280"
```

### 2. Create the IAM role and service account

```bash
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Create IAM policy
aws iam create-policy \
  --policy-name AWSLoadBalancerControllerIAMPolicy \
  --policy-document file://iam_policy.json

# Create trust policy and role
TRUST_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:aws:iam::${ACCOUNT_ID}:oidc-provider/oidc.eks.us-east-1.amazonaws.com/id/${OIDC_ID}"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "oidc.eks.us-east-1.amazonaws.com/id/${OIDC_ID}:sub": "system:serviceaccount:kube-system:aws-load-balancer-controller",
        "oidc.eks.us-east-1.amazonaws.com/id/${OIDC_ID}:aud": "sts.amazonaws.com"
      }
    }
  }]
}
EOF
)

aws iam create-role \
  --role-name AmazonEKSLoadBalancerControllerRole \
  --assume-role-policy-document "$TRUST_POLICY"

aws iam attach-role-policy \
  --role-name AmazonEKSLoadBalancerControllerRole \
  --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/AWSLoadBalancerControllerIAMPolicy

# Create annotated service account
kubectl create serviceaccount aws-load-balancer-controller -n kube-system
kubectl annotate serviceaccount aws-load-balancer-controller -n kube-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::${ACCOUNT_ID}:role/AmazonEKSLoadBalancerControllerRole
```

### 3. Tag public subnets

The ALB needs to know which subnets to use:

```bash
VPC_ID=$(aws ec2 describe-vpcs --filters "Name=tag:Name,Values=*inferflow*" \
  --query "Vpcs[0].VpcId" --output text --region us-east-1)

PUBLIC_SUBNETS=$(aws ec2 describe-subnets \
  --filters "Name=vpc-id,Values=${VPC_ID}" "Name=map-public-ip-on-launch,Values=true" \
  --query "Subnets[*].SubnetId" --output text)

for subnet in $PUBLIC_SUBNETS; do
  aws ec2 create-tags --resources $subnet --region us-east-1 \
    --tags Key=kubernetes.io/role/elb,Value=1 \
           Key=kubernetes.io/cluster/<your-cluster-name>,Value=shared
done
```

### 4. Install the controller via Helm

```bash
helm repo add eks https://aws.github.io/eks-charts && helm repo update

VPC_ID=$(aws ec2 describe-vpcs --filters "Name=tag:Name,Values=*inferflow*" \
  --query "Vpcs[0].VpcId" --output text --region us-east-1)

helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=<your-cluster-name> \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller \
  --set region=us-east-1 \
  --set vpcId=$VPC_ID
```

### 5. Deploy the Ingress

```bash
kubectl apply -f k8s/ingress.yaml
```

### 6. Get your public URL

```bash
kubectl get ingress inferflow-ingress
# ADDRESS column will show: <hash>.us-east-1.elb.amazonaws.com (takes ~2 min)
```

---

## Smoke Test

```bash
ALB=$(kubectl get ingress inferflow-ingress \
  -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

curl -X POST http://$ALB/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen/Qwen2.5-0.5B-Instruct",
    "messages": [{"role": "user", "content": "What is 2+2?"}],
    "max_tokens": 50
  }'
```

Expected response:
```json
{"choices":[{"message":{"role":"assistant","content":"2+2 is 4."}}],...}
```

---

## Quick Local Test (no ALB needed)

```bash
kubectl port-forward svc/inferflow-router 8080:80

curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen/Qwen2.5-0.5B-Instruct","messages":[{"role":"user","content":"Hello"}],"max_tokens":20}'
```
