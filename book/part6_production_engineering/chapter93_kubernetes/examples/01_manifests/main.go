// FILE: book/part6_production_engineering/chapter93_kubernetes/examples/01_manifests/main.go
// CHAPTER: 93 — Kubernetes for Go Services
// TOPIC: Kubernetes manifest structures — Deployment, Service, ConfigMap,
//        HPA, resource limits, and rolling update strategy.
//
// Run:
//   go run ./book/part6_production_engineering/chapter93_kubernetes/examples/01_manifests

package main

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// MANIFEST TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Resources struct {
	CPURequest    string
	MemoryRequest string
	CPULimit      string
	MemoryLimit   string
}

type Probe struct {
	Path                string
	Port                int
	InitialDelaySeconds int
	PeriodSeconds       int
	FailureThreshold    int
	TimeoutSeconds      int
}

type Container struct {
	Name      string
	Image     string
	Port      int
	Env       map[string]string
	Resources Resources
	Liveness  Probe
	Readiness Probe
	Startup   Probe
}

type RollingUpdate struct {
	MaxSurge       int
	MaxUnavailable int
}

type Deployment struct {
	Name      string
	Namespace string
	Replicas  int
	Labels    map[string]string
	Container Container
	Strategy  RollingUpdate
}

type Service struct {
	Name      string
	Namespace string
	Port      int
	TargetPort int
	Selector  map[string]string
}

type HPA struct {
	Name            string
	Namespace       string
	DeploymentName  string
	MinReplicas     int
	MaxReplicas     int
	CPUTargetPct    int
}

type PDB struct {
	Name         string
	Namespace    string
	MinAvailable int
	Selector     map[string]string
}

// ─────────────────────────────────────────────────────────────────────────────
// YAML RENDERING
// ─────────────────────────────────────────────────────────────────────────────

func renderDeployment(d Deployment) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: %d
      maxUnavailable: %d
  template:
    metadata:
      labels:
        app: %s
    spec:
      terminationGracePeriodSeconds: 30
      containers:
      - name: %s
        image: %s
        ports:
        - containerPort: %d
        resources:
          requests:
            cpu: %q
            memory: %q
          limits:
            cpu: %q
            memory: %q
        livenessProbe:
          httpGet:
            path: %s
            port: %d
          initialDelaySeconds: %d
          periodSeconds: %d
          failureThreshold: %d
        readinessProbe:
          httpGet:
            path: %s
            port: %d
          initialDelaySeconds: %d
          periodSeconds: %d
          failureThreshold: %d
`,
		d.Name, d.Namespace,
		d.Replicas,
		d.Labels["app"],
		d.Strategy.MaxSurge, d.Strategy.MaxUnavailable,
		d.Labels["app"],
		d.Container.Name, d.Container.Image, d.Container.Port,
		d.Container.Resources.CPURequest, d.Container.Resources.MemoryRequest,
		d.Container.Resources.CPULimit, d.Container.Resources.MemoryLimit,
		d.Container.Liveness.Path, d.Container.Liveness.Port,
		d.Container.Liveness.InitialDelaySeconds,
		d.Container.Liveness.PeriodSeconds,
		d.Container.Liveness.FailureThreshold,
		d.Container.Readiness.Path, d.Container.Readiness.Port,
		d.Container.Readiness.InitialDelaySeconds,
		d.Container.Readiness.PeriodSeconds,
		d.Container.Readiness.FailureThreshold,
	)
	if len(d.Container.Env) > 0 {
		fmt.Fprintf(&sb, "        env:\n")
		for k, v := range d.Container.Env {
			fmt.Fprintf(&sb, "        - name: %s\n          value: %q\n", k, v)
		}
	}
	return sb.String()
}

func renderService(s Service) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  type: ClusterIP
  selector:
    app: %s
  ports:
  - port: %d
    targetPort: %d
    protocol: TCP
`, s.Name, s.Namespace, s.Selector["app"], s.Port, s.TargetPort)
}

func renderHPA(h HPA) string {
	return fmt.Sprintf(`apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: %s
  namespace: %s
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: %s
  minReplicas: %d
  maxReplicas: %d
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: %d
`, h.Name, h.Namespace, h.DeploymentName, h.MinReplicas, h.MaxReplicas, h.CPUTargetPct)
}

func renderPDB(p PDB) string {
	return fmt.Sprintf(`apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: %s
  namespace: %s
spec:
  minAvailable: %d
  selector:
    matchLabels:
      app: %s
`, p.Name, p.Namespace, p.MinAvailable, p.Selector["app"])
}

// ─────────────────────────────────────────────────────────────────────────────
// RESOURCE RECOMMENDATION
// ─────────────────────────────────────────────────────────────────────────────

func recommendResources(rps, avgLatencyMs int) Resources {
	// Rule of thumb: 1 mCPU per 10 RPS; 64 Mi base + 32 Mi per 100 RPS
	cpuMilli := rps / 10
	if cpuMilli < 50 {
		cpuMilli = 50
	}
	memMi := 64 + (rps/100)*32
	if memMi < 64 {
		memMi = 64
	}
	_ = avgLatencyMs
	return Resources{
		CPURequest:    fmt.Sprintf("%dm", cpuMilli),
		MemoryRequest: fmt.Sprintf("%dMi", memMi),
		CPULimit:      fmt.Sprintf("%dm", cpuMilli*5),
		MemoryLimit:   fmt.Sprintf("%dMi", memMi*2),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 93: Kubernetes Manifest Patterns ===")
	fmt.Println()

	// ── BUILD MANIFEST SET ────────────────────────────────────────────────────
	app := "order-service"
	ns := "production"
	labels := map[string]string{"app": app}

	res := recommendResources(500, 20)
	fmt.Printf("--- Resource recommendations (500 RPS, 20ms latency) ---\n")
	fmt.Printf("  CPU request=%s limit=%s\n", res.CPURequest, res.CPULimit)
	fmt.Printf("  Mem request=%s limit=%s\n", res.MemoryRequest, res.MemoryLimit)
	fmt.Println()

	d := Deployment{
		Name: app, Namespace: ns, Replicas: 3, Labels: labels,
		Strategy: RollingUpdate{MaxSurge: 1, MaxUnavailable: 0},
		Container: Container{
			Name:      app,
			Image:     "registry.example.com/" + app + ":1.2.3",
			Port:      8080,
			Resources: res,
			Env:       map[string]string{"APP_ENV": "production", "LOG_LEVEL": "info"},
			Liveness:  Probe{Path: "/healthz", Port: 8080, InitialDelaySeconds: 5, PeriodSeconds: 10, FailureThreshold: 3},
			Readiness: Probe{Path: "/readyz", Port: 8080, InitialDelaySeconds: 5, PeriodSeconds: 5, FailureThreshold: 2},
		},
	}

	svc := Service{Name: app, Namespace: ns, Port: 80, TargetPort: 8080, Selector: labels}
	hpa := HPA{Name: app + "-hpa", Namespace: ns, DeploymentName: app, MinReplicas: 2, MaxReplicas: 10, CPUTargetPct: 70}
	pdb := PDB{Name: app + "-pdb", Namespace: ns, MinAvailable: 1, Selector: labels}

	fmt.Println("--- Deployment manifest ---")
	fmt.Println(renderDeployment(d))

	fmt.Println("--- Service manifest ---")
	fmt.Println(renderService(svc))

	fmt.Println("--- HPA manifest ---")
	fmt.Println(renderHPA(hpa))

	fmt.Println("--- PodDisruptionBudget manifest ---")
	fmt.Println(renderPDB(pdb))

	// ── KEY CONCEPTS ──────────────────────────────────────────────────────────
	fmt.Println("--- Key concepts ---")
	fmt.Println(`  requests vs limits:
    requests  = what scheduler uses to place pod on a node
    limits    = max allowed; CPU exceeded → throttled; memory exceeded → OOMKilled

  RollingUpdate: maxSurge=1, maxUnavailable=0
    → zero-downtime: new pod must be Ready before old one terminates

  HPA formula: desiredReplicas = ceil(current × currentMetric / targetMetric)
    e.g. 3 replicas at 85% CPU, target 70% → ceil(3 × 85/70) = ceil(3.64) = 4

  PodDisruptionBudget:
    minAvailable: 1 → at least 1 pod always available during node drain / rolling update`)
}
