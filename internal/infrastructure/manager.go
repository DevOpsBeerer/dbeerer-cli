package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	PlaygroundRepoURL = "https://github.com/DevOpsBeerer/playground.git"
	TempDirPrefix     = "devopsbeerer-infra-"
)

// Manager handles infrastructure operations
type Manager struct {
	workDir string
}

// NewManager creates a new infrastructure manager
func NewManager() *Manager {
	return &Manager{}
}

// DeployInfrastructure clones the playground repo and runs setup scripts
func (m *Manager) DeployInfrastructure() error {
	fmt.Println("🍺 Starting infrastructure deployment...")

	// Create temporary working directory
	tempDir, err := os.MkdirTemp("", TempDirPrefix)
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	m.workDir = tempDir

	fmt.Printf("📁 Working directory: %s\n", m.workDir)

	// Clone the playground repository
	if err := m.cloneRepository(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Install K3s
	if err := m.installK3s(); err != nil {
		return fmt.Errorf("failed to install K3s: %w", err)
	}

	// Initialize K3s with components
	if err := m.initializeK3s(); err != nil {
		return fmt.Errorf("failed to initialize K3s: %w", err)
	}

	fmt.Println("✅ Infrastructure deployed successfully!")
	fmt.Printf("🗑️  Cleaning up temporary files...\n")

	// Clean up temporary directory
	if err := os.RemoveAll(m.workDir); err != nil {
		fmt.Printf("⚠️  Warning: failed to clean up temp directory: %v\n", err)
	}

	return nil
}

// cloneRepository clones the playground repository
func (m *Manager) cloneRepository() error {
	fmt.Printf("📥 Cloning playground repository...\n")

	repoDir := filepath.Join(m.workDir, "playground")

	cmd := exec.Command("git", "clone", PlaygroundRepoURL, repoDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	fmt.Printf("✅ Repository cloned to %s\n", repoDir)
	return nil
}

// installK3s runs the install-k3s.sh script
func (m *Manager) installK3s() error {
	fmt.Printf("🚀 Installing K3s...\n")

	scriptPath := filepath.Join(m.workDir, "playground", "install-k3s.sh")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("install-k3s.sh script not found at %s", scriptPath)
	}

	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	// Run the script
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = filepath.Join(m.workDir, "playground")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install-k3s.sh execution failed: %w", err)
	}

	fmt.Printf("✅ K3s installed successfully\n")
	return nil
}

// initializeK3s runs the init-k3s.sh script
func (m *Manager) initializeK3s() error {
	fmt.Printf("⚙️  Initializing K3s with components (cert-manager, SSO, ingress controller)...\n")

	scriptPath := filepath.Join(m.workDir, "playground", "init-k3s.sh")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("init-k3s.sh script not found at %s", scriptPath)
	}

	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	// Run the script
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = filepath.Join(m.workDir, "playground")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("init-k3s.sh execution failed: %w", err)
	}

	fmt.Printf("✅ K3s initialized with all components\n")
	return nil
}

// CheckInfrastructure checks if infrastructure components are running
func (m *Manager) CheckInfrastructure() (*InfrastructureStatus, error) {
	status := &InfrastructureStatus{}

	// Check if kubectl is available
	if err := exec.Command("kubectl", "version", "--client").Run(); err != nil {
		status.KubectlAvailable = false
		return status, nil
	}
	status.KubectlAvailable = true

	// Check K3s cluster
	if err := exec.Command("kubectl", "cluster-info").Run(); err != nil {
		status.ClusterRunning = false
	} else {
		status.ClusterRunning = true
	}

	// Check components
	status.Components = m.checkComponents()

	return status, nil
}

// checkComponents checks individual infrastructure components
func (m *Manager) checkComponents() map[string]bool {
	components := map[string]bool{
		"cert-manager":       false,
		"ingress-controller": false,
		"keycloak":           false,
	}

	// Define component configurations
	componentConfigs := []struct {
		name        string
		helmRelease string
		namespace   string
		selector    string // optional label selector for pods
	}{
		{
			name:        "cert-manager",
			helmRelease: "cert-manager",
			namespace:   "cert-manager",
		},
		{
			name:        "ingress-controller",
			helmRelease: "ingress-nginx",
			namespace:   "ingress-nginx",
			selector:    "app.kubernetes.io/name=ingress-nginx",
		},
		{
			name:        "keycloak",
			helmRelease: "sso",
			namespace:   "sso",
		},
	}

	for _, config := range componentConfigs {
		components[config.name] = m.isComponentHealthy(config.helmRelease, config.namespace, config.selector)
	}

	return components
}

func (m *Manager) isComponentHealthy(helmRelease, namespace, selector string) bool {
	// First check Helm release status
	if !m.isHelmReleaseDeployed(helmRelease, namespace) {
		return false
	}

	// Then check pod readiness
	return m.arePodsReady(namespace, selector)
}

func (m *Manager) isHelmReleaseDeployed(releaseName, namespace string) bool {
	cmd := exec.Command("helm", "status", releaseName, "-n", namespace, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse Helm status
	var status struct {
		Info struct {
			Status string `json:"status"`
		} `json:"info"`
	}

	if err := json.Unmarshal(output, &status); err != nil {
		return false
	}

	return status.Info.Status == "deployed"
}

func (m *Manager) arePodsReady(namespace, selector string) bool {
	args := []string{"get", "pods", "-n", namespace, "-o", "json"}
	if selector != "" {
		args = append(args, "-l", selector)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse pod list
	var podList struct {
		Items []struct {
			Status struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
				ContainerStatuses []struct {
					Ready bool `json:"ready"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &podList); err != nil {
		return false
	}

	// Check if we have any pods
	if len(podList.Items) == 0 {
		return false
	}

	// Check if all pods are ready
	for _, pod := range podList.Items {
		// Check container readiness
		for _, container := range pod.Status.ContainerStatuses {
			if !container.Ready {
				return false
			}
		}

		// Check pod Ready condition
		podReady := false
		for _, condition := range pod.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				podReady = true
				break
			}
		}
		if !podReady {
			return false
		}
	}

	return true
}

// InfrastructureStatus represents the status of infrastructure components
type InfrastructureStatus struct {
	KubectlAvailable bool
	ClusterRunning   bool
	Components       map[string]bool
}
