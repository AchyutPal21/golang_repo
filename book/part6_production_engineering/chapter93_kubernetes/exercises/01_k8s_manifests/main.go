// FILE: book/part6_production_engineering/chapter93_kubernetes/exercises/01_k8s_manifests/main.go
// CHAPTER: 93 — Kubernetes for Go Services
// EXERCISE: Generate a complete production-grade manifest set for a Go service —
//           Deployment, Service, HPA, PDB, ConfigMap, and Secret reference.
//
// Run:
//   go run ./book/part6_production_engineering/chapter93_kubernetes/exercises/01_k8s_manifests

package main

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE PROFILE — drives manifest generation
// ─────────────────────────────────────────────────────────────────────────────

type ServiceProfile struct {
	Name      string
	Namespace string
	Image     string
	Version   string
	Port      int
	RPS       int // expected requests per second
	StartupMs int // startup time in milliseconds
	Env       map[string]string
	Secrets   []string // names of secrets to mount from K8s Secrets
}

// ─────────────────────────────────────────────────────────────────────────────
// RESOURCE CALCULATOR
// ─────────────────────────────────────────────────────────────────────────────

type ResourceRecommendation struct {
	CPURequestM    int // millicores
	MemRequestMi   int // mebibytes
	CPULimitM      int
	MemLimitMi     int
}

func calcResources(rps int) ResourceRecommendation {
	cpuReq := int(math.Max(50, float64(rps)/10))
	memReq := int(math.Max(64, float64(64+rps/100*32)))
	return ResourceRecommendation{
		CPURequestM:  cpuReq,
		MemRequestMi: memReq,
		CPULimitM:    cpuReq * 5,
		MemLimitMi:   memReq * 2,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PROBE TIMING CALCULATOR
// ─────────────────────────────────────────────────────────────────────────────

type ProbeTimings struct {
	StartupInitialDelay int
	StartupPeriod       int
	StartupFailures     int
	LivenessPeriod      int
	ReadinessPeriod     int
}

func calcProbes(startupMs int) ProbeTimings {
	startupSecs := startupMs/1000 + 5 // add 5s buffer
	return ProbeTimings{
		StartupInitialDelay: 0,
		StartupPeriod:       2,
		StartupFailures:     (startupSecs + 1) / 2, // ceil(startupSecs / period)
		LivenessPeriod:      10,
		ReadinessPeriod:     5,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MANIFEST GENERATORS
// ─────────────────────────────────────────────────────────────────────────────

func genConfigMap(p ServiceProfile) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: %s-config\n  namespace: %s\ndata:\n", p.Name, p.Namespace)
	for k, v := range p.Env {
		fmt.Fprintf(&sb, "  %s: %q\n", k, v)
	}
	return sb.String()
}

func genDeployment(p ServiceProfile, res ResourceRecommendation, probes ProbeTimings) string {
	var sb strings.Builder

	envFrom := ""
	if len(p.Env) > 0 {
		envFrom = fmt.Sprintf(`
        envFrom:
        - configMapRef:
            name: %s-config`, p.Name)
	}

	secretEnv := ""
	for _, s := range p.Secrets {
		secretEnv += fmt.Sprintf(`
        - name: %s
          valueFrom:
            secretKeyRef:
              name: %s-secrets
              key: %s`, strings.ToUpper(s), p.Name, s)
	}
	if secretEnv != "" {
		secretEnv = "\n        env:" + secretEnv
	}

	fmt.Fprintf(&sb, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    version: %s
spec:
  replicas: 2
  selector:
    matchLabels:
      app: %s
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: %s
        version: %s
    spec:
      terminationGracePeriodSeconds: 30
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
      containers:
      - name: %s
        image: %s:%s
        ports:
        - containerPort: %d%s%s
        resources:
          requests:
            cpu: %dm
            memory: %dMi
          limits:
            cpu: %dm
            memory: %dMi
        startupProbe:
          httpGet:
            path: /readyz
            port: %d
          initialDelaySeconds: %d
          periodSeconds: %d
          failureThreshold: %d
        livenessProbe:
          httpGet:
            path: /healthz
            port: %d
          periodSeconds: %d
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: %d
          periodSeconds: %d
          failureThreshold: 2
`,
		p.Name, p.Namespace, p.Name, p.Version,
		p.Name,
		p.Name, p.Version,
		p.Name, p.Image, p.Version, p.Port,
		envFrom, secretEnv,
		res.CPURequestM, res.MemRequestMi,
		res.CPULimitM, res.MemLimitMi,
		p.Port, probes.StartupInitialDelay, probes.StartupPeriod, probes.StartupFailures,
		p.Port, probes.LivenessPeriod,
		p.Port, probes.ReadinessPeriod,
	)
	return sb.String()
}

func genService(p ServiceProfile) string {
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
  - port: 80
    targetPort: %d
`, p.Name, p.Namespace, p.Name, p.Port)
}

func genHPA(p ServiceProfile) string {
	minR := 2
	maxR := int(math.Max(float64(minR), float64(p.RPS)/200))
	if maxR < 4 {
		maxR = 4
	}
	return fmt.Sprintf(`apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: %s-hpa
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
        averageUtilization: 70
`, p.Name, p.Namespace, p.Name, minR, maxR)
}

func genPDB(p ServiceProfile) string {
	return fmt.Sprintf(`apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: %s-pdb
  namespace: %s
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: %s
`, p.Name, p.Namespace, p.Name)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 93 Exercise: K8s Manifest Generator ===")
	fmt.Println()

	profile := ServiceProfile{
		Name:      "payment-service",
		Namespace: "production",
		Image:     "registry.example.com/payment-service",
		Version:   "2.4.1",
		Port:      8080,
		RPS:       300,
		StartupMs: 8000,
		Env: map[string]string{
			"APP_ENV":   "production",
			"LOG_LEVEL": "info",
			"PORT":      "8080",
		},
		Secrets: []string{"database_url", "stripe_api_key"},
	}

	res := calcResources(profile.RPS)
	probes := calcProbes(profile.StartupMs)

	fmt.Printf("--- Profile: %s (RPS=%d, startup=%dms) ---\n", profile.Name, profile.RPS, profile.StartupMs)
	fmt.Printf("  Resources: cpu=%dm/%dm  mem=%dMi/%dMi\n",
		res.CPURequestM, res.CPULimitM, res.MemRequestMi, res.MemLimitMi)
	fmt.Printf("  Startup probe: %d checks × %ds period = %ds window\n",
		probes.StartupFailures, probes.StartupPeriod, probes.StartupFailures*probes.StartupPeriod)
	fmt.Println()

	// ── GENERATE AND PRINT ALL MANIFESTS ──────────────────────────────────────
	separator := "---\n"
	manifests := []struct {
		name    string
		content string
	}{
		{"ConfigMap", genConfigMap(profile)},
		{"Deployment", genDeployment(profile, res, probes)},
		{"Service", genService(profile)},
		{"HPA", genHPA(profile)},
		{"PDB", genPDB(profile)},
	}

	for _, m := range manifests {
		fmt.Printf("--- %s ---\n", m.name)
		fmt.Print(m.content)
		fmt.Print(separator)
	}

	// ── APPLY INSTRUCTIONS ────────────────────────────────────────────────────
	fmt.Println("--- Apply to cluster ---")
	fmt.Printf(`  # Save manifests (in a real project, write to deploy/%s.yaml)
  kubectl apply -f deploy/%s.yaml
  kubectl rollout status deployment/%s -n %s

  # Verify HPA
  kubectl get hpa %s-hpa -n %s

  # Watch pod rollout
  kubectl get pods -l app=%s -n %s -w
`, profile.Name, profile.Name, profile.Name, profile.Namespace,
		profile.Name, profile.Namespace,
		profile.Name, profile.Namespace)

	_ = os.Stdout
}
