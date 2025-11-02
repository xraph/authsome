# AuthSome Cloud Deployment Guide

**Infrastructure setup and deployment instructions for control plane operators**

## Table of Contents

- [Prerequisites](#prerequisites)
- [Infrastructure Setup](#infrastructure-setup)
- [Control Plane Deployment](#control-plane-deployment)
- [Monitoring Setup](#monitoring-setup)
- [Security Configuration](#security-configuration)
- [Production Checklist](#production-checklist)
- [Maintenance](#maintenance)

## Prerequisites

### Required Tools

```bash
# Kubernetes cluster (1.25+)
kubectl version --client

# Helm package manager (3.10+)
helm version

# Terraform (1.5+) for infrastructure provisioning
terraform version

# Docker for local development
docker version

# Go (1.21+) for building control plane
go version
```

### Required Services

- **Kubernetes Cluster**: EKS, GKE, or AKS (3+ nodes, 8GB+ RAM each)
- **PostgreSQL**: RDS/Cloud SQL for control plane (13+)
- **Redis**: ElastiCache/MemoryStore for caching (6+)
- **NATS**: Message queue for async operations
- **Object Storage**: S3/GCS for backups
- **Domain**: For api.authsome.cloud, dashboard.authsome.cloud
- **TLS Certificates**: Let's Encrypt or commercial

### Infrastructure Budget Estimate

```
Minimum Production Setup:
- Kubernetes (3 nodes): $300-500/month
- Control Plane Database: $50-100/month
- Redis: $30-50/month
- Load Balancer: $20/month
- Monitoring Stack: $50-100/month
- Total: ~$500-800/month (before customer workloads)

Enterprise Setup:
- Kubernetes (10+ nodes): $1500+/month
- Multi-region: 2-3x base cost
```

## Infrastructure Setup

### 1. Kubernetes Cluster (AWS EKS Example)

```bash
# terraform/eks/main.tf
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  
  name = "authsome-cloud-vpc"
  cidr = "10.0.0.0/16"
  
  azs             = ["${var.region}a", "${var.region}b", "${var.region}c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
  
  enable_nat_gateway   = true
  enable_dns_hostnames = true
  
  tags = {
    "kubernetes.io/cluster/authsome-cloud" = "shared"
  }
}

module "eks" {
  source = "terraform-aws-modules/eks/aws"
  
  cluster_name    = "authsome-cloud"
  cluster_version = "1.28"
  
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets
  
  eks_managed_node_groups = {
    # System workloads (control plane, monitoring)
    system = {
      name           = "system-nodes"
      instance_types = ["t3.xlarge"]
      min_size       = 3
      max_size       = 10
      desired_size   = 3
      
      labels = {
        role = "system"
      }
      
      taints = [{
        key    = "dedicated"
        value  = "system"
        effect = "NoSchedule"
      }]
    }
    
    # Customer application workloads
    apps = {
      name           = "app-nodes"
      instance_types = ["t3.2xlarge"]
      min_size       = 5
      max_size       = 50
      desired_size   = 5
      
      labels = {
        role = "apps"
      }
    }
  }
  
  # Enable IRSA (IAM Roles for Service Accounts)
  enable_irsa = true
}

# RDS PostgreSQL for control plane
resource "aws_db_instance" "control_plane" {
  identifier     = "authsome-control-plane"
  engine         = "postgres"
  engine_version = "15.4"
  instance_class = "db.t3.medium"
  
  allocated_storage     = 100
  max_allocated_storage = 500
  storage_encrypted     = true
  
  db_name  = "authsome_control"
  username = "authsome"
  password = random_password.db_password.result
  
  multi_az               = true
  backup_retention_period = 30
  backup_window          = "03:00-04:00"
  maintenance_window     = "mon:04:00-mon:05:00"
  
  vpc_security_group_ids = [aws_security_group.db.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name
  
  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  
  tags = {
    Name = "authsome-control-plane"
  }
}

# ElastiCache Redis
resource "aws_elasticache_replication_group" "cache" {
  replication_group_id       = "authsome-cache"
  replication_group_description = "AuthSome Cloud cache"
  
  engine               = "redis"
  engine_version       = "7.0"
  node_type            = "cache.t3.medium"
  number_cache_clusters = 2
  
  parameter_group_name = "default.redis7"
  port                 = 6379
  
  subnet_group_name  = aws_elasticache_subnet_group.main.name
  security_group_ids = [aws_security_group.redis.id]
  
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  
  automatic_failover_enabled = true
  multi_az_enabled          = true
  
  snapshot_retention_limit = 7
  snapshot_window         = "03:00-05:00"
  
  tags = {
    Name = "authsome-cache"
  }
}
```

**Deploy Infrastructure:**
```bash
cd terraform/eks
terraform init
terraform plan
terraform apply
```

### 2. Configure kubectl

```bash
# Get EKS cluster credentials
aws eks update-kubeconfig --region us-east-1 --name authsome-cloud

# Verify connection
kubectl get nodes
```

### 3. Install Core Dependencies

```bash
# Create namespaces
kubectl create namespace authsome-system
kubectl create namespace authsome-apps
kubectl create namespace monitoring

# Install cert-manager (for TLS)
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true

# Install NATS
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm install nats nats/nats \
  --namespace authsome-system \
  --set nats.jetstream.enabled=true \
  --set nats.jetstream.memStorage.enabled=true \
  --set nats.jetstream.fileStorage.enabled=true \
  --set nats.jetstream.fileStorage.size=10Gi

# Install ingress-nginx
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.service.type=LoadBalancer \
  --set controller.metrics.enabled=true
```

## Control Plane Deployment

### 1. Build Control Plane Images

```bash
# Clone repository
git clone https://github.com/xraph/authsome-cloud
cd authsome-cloud

# Build images
docker build -t authsome/control-plane:latest -f cmd/control-plane/Dockerfile .
docker build -t authsome/proxy:latest -f cmd/proxy/Dockerfile .
docker build -t authsome/provisioner:latest -f cmd/provisioner/Dockerfile .
docker build -t authsome/dashboard:latest -f cmd/dashboard/Dockerfile .

# Push to registry (replace with your registry)
docker tag authsome/control-plane:latest your-registry/authsome/control-plane:v1.0.0
docker push your-registry/authsome/control-plane:v1.0.0

docker tag authsome/proxy:latest your-registry/authsome/proxy:v1.0.0
docker push your-registry/authsome/proxy:v1.0.0

docker tag authsome/provisioner:latest your-registry/authsome/provisioner:v1.0.0
docker push your-registry/authsome/provisioner:v1.0.0

docker tag authsome/dashboard:latest your-registry/authsome/dashboard:v1.0.0
docker push your-registry/authsome/dashboard:v1.0.0
```

### 2. Configure Secrets

```bash
# Create secret for control plane database
kubectl create secret generic control-plane-db \
  --namespace authsome-system \
  --from-literal=url="postgres://user:pass@host:5432/authsome_control?sslmode=require"

# Create secret for Redis
kubectl create secret generic control-plane-redis \
  --namespace authsome-system \
  --from-literal=url="redis://host:6379/0"

# Create secret for JWT signing
kubectl create secret generic control-plane-jwt \
  --namespace authsome-system \
  --from-literal=secret="$(openssl rand -base64 32)"

# Create secret for API key encryption
kubectl create secret generic control-plane-encryption \
  --namespace authsome-system \
  --from-literal=key="$(openssl rand -base64 32)"
```

### 3. Deploy Control Plane API

```yaml
# k8s/control-plane/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: control-plane
  namespace: authsome-system
  labels:
    app: control-plane
spec:
  replicas: 3
  selector:
    matchLabels:
      app: control-plane
  template:
    metadata:
      labels:
        app: control-plane
    spec:
      nodeSelector:
        role: system
      tolerations:
      - key: dedicated
        operator: Equal
        value: system
        effect: NoSchedule
      containers:
      - name: control-plane
        image: your-registry/authsome/control-plane:v1.0.0
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: control-plane-db
              key: url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: control-plane-redis
              key: url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: control-plane-jwt
              key: secret
        - name: ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: control-plane-encryption
              key: key
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: ENVIRONMENT
          value: "production"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: control-plane
  namespace: authsome-system
spec:
  selector:
    app: control-plane
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: control-plane
  namespace: authsome-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: control-plane
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

**Deploy:**
```bash
kubectl apply -f k8s/control-plane/
```

### 4. Deploy Proxy Service

```yaml
# k8s/proxy/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: proxy
  namespace: authsome-system
spec:
  replicas: 5
  selector:
    matchLabels:
      app: proxy
  template:
    metadata:
      labels:
        app: proxy
    spec:
      nodeSelector:
        role: system
      tolerations:
      - key: dedicated
        operator: Equal
        value: system
        effect: NoSchedule
      containers:
      - name: proxy
        image: your-registry/authsome/proxy:v1.0.0
        ports:
        - containerPort: 8081
          name: http
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: control-plane-db
              key: url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: control-plane-redis
              key: url
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 5
        readinessProbe:
          httpGet:
            path: /ready
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 3
---
apiVersion: v1
kind: Service
metadata:
  name: proxy
  namespace: authsome-system
spec:
  selector:
    app: proxy
  ports:
  - protocol: TCP
    port: 8081
    targetPort: 8081
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: proxy
  namespace: authsome-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: proxy
  minReplicas: 5
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 60
```

**Deploy:**
```bash
kubectl apply -f k8s/proxy/
```

### 5. Deploy Provisioner Workers

```yaml
# k8s/provisioner/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: provisioner
  namespace: authsome-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: provisioner
  template:
    metadata:
      labels:
        app: provisioner
    spec:
      serviceAccountName: provisioner
      nodeSelector:
        role: system
      tolerations:
      - key: dedicated
        operator: Equal
        value: system
        effect: NoSchedule
      containers:
      - name: provisioner
        image: your-registry/authsome/provisioner:v1.0.0
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: control-plane-db
              key: url
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
---
# ServiceAccount with permissions to create namespaces, deployments, etc.
apiVersion: v1
kind: ServiceAccount
metadata:
  name: provisioner
  namespace: authsome-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: provisioner
rules:
- apiGroups: [""]
  resources: ["namespaces", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "create", "update", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "create", "update", "delete"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list", "create", "update", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: provisioner
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: provisioner
subjects:
- kind: ServiceAccount
  name: provisioner
  namespace: authsome-system
```

**Deploy:**
```bash
kubectl apply -f k8s/provisioner/
```

### 6. Deploy Dashboard

```yaml
# k8s/dashboard/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dashboard
  namespace: authsome-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: dashboard
  template:
    metadata:
      labels:
        app: dashboard
    spec:
      nodeSelector:
        role: system
      containers:
      - name: dashboard
        image: your-registry/authsome/dashboard:v1.0.0
        ports:
        - containerPort: 3000
          name: http
        env:
        - name: CONTROL_PLANE_API_URL
          value: "http://control-plane:8080"
        - name: NODE_ENV
          value: "production"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: dashboard
  namespace: authsome-system
spec:
  selector:
    app: dashboard
  ports:
  - protocol: TCP
    port: 3000
    targetPort: 3000
  type: ClusterIP
```

**Deploy:**
```bash
kubectl apply -f k8s/dashboard/
```

### 7. Configure Ingress

```yaml
# k8s/ingress/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: authsome-cloud
  namespace: authsome-system
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - api.authsome.cloud
    - dashboard.authsome.cloud
    secretName: authsome-cloud-tls
  rules:
  # Control Plane API
  - host: api.authsome.cloud
    http:
      paths:
      - path: /control
        pathType: Prefix
        backend:
          service:
            name: control-plane
            port:
              number: 8080
      # Customer API Proxy
      - path: /v1
        pathType: Prefix
        backend:
          service:
            name: proxy
            port:
              number: 8081
  # Dashboard
  - host: dashboard.authsome.cloud
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: dashboard
            port:
              number: 3000
```

**Deploy:**
```bash
kubectl apply -f k8s/ingress/
```

## Monitoring Setup

### 1. Install Prometheus Stack

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install kube-prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set prometheus.prometheusSpec.retention=30d \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=100Gi
```

### 2. Configure ServiceMonitors

```yaml
# k8s/monitoring/servicemonitors.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: control-plane
  namespace: authsome-system
spec:
  selector:
    matchLabels:
      app: control-plane
  endpoints:
  - port: http
    path: /metrics
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: proxy
  namespace: authsome-system
spec:
  selector:
    matchLabels:
      app: proxy
  endpoints:
  - port: http
    path: /metrics
```

### 3. Create Dashboards

Import pre-built Grafana dashboards from `k8s/monitoring/dashboards/`.

### 4. Configure Alerts

```yaml
# k8s/monitoring/alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: authsome-cloud-alerts
  namespace: monitoring
spec:
  groups:
  - name: authsome-cloud
    interval: 30s
    rules:
    - alert: ControlPlaneDown
      expr: up{job="control-plane"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Control Plane API is down"
    
    - alert: ProxyHighLatency
      expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{job="proxy"}[5m])) > 1
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Proxy P95 latency > 1s"
    
    - alert: ProvisioningBacklog
      expr: nats_jetstream_consumer_num_pending{consumer="provisioners"} > 100
      for: 30m
      labels:
        severity: warning
      annotations:
        summary: "Provisioning queue has {{ $value }} pending jobs"
```

## Security Configuration

### 1. Network Policies

```yaml
# k8s/security/network-policies.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-ingress
  namespace: authsome-apps
spec:
  podSelector: {}
  policyTypes:
  - Ingress
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-proxy-to-apps
  namespace: authsome-apps
spec:
  podSelector:
    matchLabels:
      app: authsome
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: authsome-system
      podSelector:
        matchLabels:
          app: proxy
```

### 2. Pod Security Standards

```yaml
# k8s/security/pod-security.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: authsome-apps
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### 3. Secrets Encryption at Rest

Enable encryption in Kubernetes:
```yaml
# /etc/kubernetes/enc/encryption-config.yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    - aescbc:
        keys:
        - name: key1
          secret: <base64-encoded-secret>
    - identity: {}
```

## Production Checklist

### Pre-Launch

- [ ] SSL certificates configured (TLS 1.3)
- [ ] Database backups automated (daily + PITR)
- [ ] Monitoring dashboards created
- [ ] Alerting configured (PagerDuty/Slack)
- [ ] Log aggregation setup (ELK/Loki)
- [ ] Rate limiting configured
- [ ] DDoS protection enabled (Cloudflare)
- [ ] Security scanning (Trivy/Snyk)
- [ ] Load testing completed (k6/Gatling)
- [ ] Disaster recovery plan documented
- [ ] Incident response runbook created

### Post-Launch

- [ ] Status page setup (status.authsome.cloud)
- [ ] Customer documentation published
- [ ] Support channels established
- [ ] Billing integration tested
- [ ] Compliance audits scheduled
- [ ] Performance baselines established
- [ ] Backup restoration tested

## Maintenance

### Routine Operations

**Daily:**
- Monitor dashboard for anomalies
- Check alert queue
- Review error logs

**Weekly:**
- Review usage trends
- Check for security updates
- Review capacity planning metrics

**Monthly:**
- Database vacuum and analyze
- Review and rotate logs
- Disaster recovery drill
- Security patch updates

### Scaling Operations

```bash
# Scale control plane
kubectl scale deployment control-plane -n authsome-system --replicas=10

# Scale proxy
kubectl scale deployment proxy -n authsome-system --replicas=20

# Add node pool
# (varies by cloud provider)
```

### Backup and Recovery

```bash
# Backup control plane database
pg_dump -h $DB_HOST -U authsome authsome_control | \
  gzip > backup-$(date +%Y%m%d).sql.gz

# Upload to S3
aws s3 cp backup-$(date +%Y%m%d).sql.gz s3://authsome-backups/

# Restore from backup
gunzip < backup-20251101.sql.gz | \
  psql -h $DB_HOST -U authsome authsome_control
```

---

**Next:** [Billing Integration Guide](./BILLING.md)

