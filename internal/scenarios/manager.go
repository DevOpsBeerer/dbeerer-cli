package scenarios

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"path/filepath"

	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	MetadataURL    = "https://raw.githubusercontent.com/DevOpsBeerer/playground-scenarios-charts/refs/heads/main/metadata.json"
	RequestTimeout = 10 * time.Second
	ReleaseName    = "devopsbeerer-scenario"
	ChartBaseURL   = "https://raw.githubusercontent.com/DevOpsBeerer/playground-scenarios-charts/refs/heads/main"
)

// getHelmReleaseName returns the Helm release name for a scenario
// Using scenario ID ensures unique release names and allows multiple scenarios
// to be installed in different namespaces during development/testing
func getHelmReleaseName(scenarioID string) string {
	return fmt.Sprintf("devopsbeerer-%s", scenarioID)
}

// getHelmNamespace returns the namespace for a scenario
// Each scenario gets its own namespace for isolation
func getHelmNamespace(scenarioID string) string {
	return fmt.Sprintf("devopsbeerer-%s", scenarioID)
}

// Scenario represents a single scenario from metadata
type Scenario struct {
	Name        string   `json:"name"`
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Features    []string `json:"features"`
	HelmChart   struct {
		Link string `json:"link"`
		Dir  string `json:"dir"`
	} `json:"helmChart"`
}

// Manager handles scenario operations
type Manager struct {
	httpClient    *http.Client
	settings      *cli.EnvSettings
	namespace     string
	dynamicClient dynamic.Interface
	gvr           schema.GroupVersionResource
}

// ActiveScenarioInfo contains information about the active scenario
type ActiveScenarioInfo struct {
	ScenarioID   string
	ScenarioName string
	Phase        string
}

// ScenarioStatus represents the status of an active scenario
type ScenarioStatus struct {
	ScenarioID  string
	Phase       string
	Message     string
	HelmRelease string
	StartTime   string
	HelmStatus  string
}

// NewManager creates a new scenario manager
func NewManager() (*Manager, error) {
	settings := cli.New()

	// Build kubeconfig path
	var kubeconfig string = filepath.Join("/etc/rancher/k3s", "k3s.yaml")

	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Define GVR for ScenarioDefinition
	gvr := schema.GroupVersionResource{
		Group:    "devopsbeerer.io",
		Version:  "v1alpha1",
		Resource: "scenariodefinitions",
	}

	return &Manager{
		dynamicClient: dynamicClient,
		gvr:           gvr,
		settings:      settings,
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}, nil

}

// InstallScenario installs a scenario using Helm
func (m *Manager) InstallScenario(scenarioID string) error {
	fmt.Printf("🔍 Checking if scenario exists: %s\n", scenarioID)

	// First, verify the scenario exists
	scenario, err := m.GetScenario(scenarioID)
	if err != nil {
		return fmt.Errorf("scenario '%s' not found: %w", scenarioID, err)
	}

	fmt.Printf("✅ Found scenario: %s\n", scenario.Name)

	// Check for existing active scenario
	fmt.Printf("🔍 Checking for existing scenario deployment...\n")

	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.io",
		Version:  "v1alpha1",
		Resource: "activescenarios",
	}

	// Try to get existing active scenario
	existing, err := m.dynamicClient.Resource(activeGVR).
		Get(context.TODO(), "current-playground-scenario", metav1.GetOptions{})

	if err == nil && existing != nil {
		// Get the current scenario ID
		currentID, _, _ := unstructured.NestedString(existing.Object, "spec", "scenarioId")
		if currentID == scenarioID {
			fmt.Printf("✅ Scenario '%s' is already active\n", scenarioID)
			return nil
		}

		fmt.Printf("🔄 Switching from scenario '%s' to '%s'\n", currentID, scenarioID)

		// Uninstall existing scenario
		if err := m.UninstallScenario(); err != nil {
			// Log but don't fail if uninstall fails
			fmt.Printf("⚠️  Warning: %v\n", err)
		}
	}

	// Create the ActiveScenario CRD first
	fmt.Printf("📝 Creating ActiveScenario resource...\n")

	activeScenario := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "devopsbeerer.io/v1alpha1",
			"kind":       "ActiveScenario",
			"metadata": map[string]interface{}{
				"name": "current-playground-scenario",
			},
			"spec": map[string]interface{}{
				"scenarioId": scenarioID,
			},
		},
	}

	// Create the ActiveScenario
	_, err = m.dynamicClient.Resource(activeGVR).
		Create(context.TODO(), activeScenario, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create active scenario: %w", err)
	}

	// Update ActiveScenario status
	if err := m.UpdateActiveScenarioStatus(scenarioID, "Pending", getHelmReleaseName(scenarioID)); err != nil {
		fmt.Printf("⚠️  Warning: failed to update status: %v\n", err)
	}

	// Download and install Helm chart
	fmt.Printf("📥 Downloading chart for scenario: %s\n", scenarioID)

	// Download chart from GitHub
	chartPath, err := m.downloadChart(scenario)
	if err != nil {
		// If download fails, clean up the CRD
		m.dynamicClient.Resource(activeGVR).
			Delete(context.TODO(), "current-playground-scenario", metav1.DeleteOptions{})
		return fmt.Errorf("failed to download chart: %w", err)
	}

	fmt.Printf("📦 Installing scenario via Helm...\n")

	// Update ActiveScenario status
	if err := m.UpdateActiveScenarioStatus(scenarioID, "Deploying", getHelmReleaseName(scenarioID)); err != nil {
		fmt.Printf("⚠️  Warning: failed to update status: %v\n", err)
	}

	// Install the chart
	if err := m.installChart(chartPath, scenarioID); err != nil {
		// If install fails, clean up the CRD
		m.dynamicClient.Resource(activeGVR).
			Delete(context.TODO(), "current-playground-scenario", metav1.DeleteOptions{})
		return fmt.Errorf("failed to install chart: %w", err)
	}
	defer os.RemoveAll(chartPath) // Clean up temp files

	// Update ActiveScenario status
	if err := m.UpdateActiveScenarioStatus(scenarioID, "Running", getHelmReleaseName(scenarioID)); err != nil {
		fmt.Printf("⚠️  Warning: failed to update status: %v\n", err)
	}

	fmt.Printf("🎉 Scenario '%s' installed successfully!\n", scenario.Name)
	m.showScenarioInfo(scenario)

	return nil
}

// UninstallScenario removes the current scenario deployment
func (m *Manager) UninstallScenario() error {
	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.io",
		Version:  "v1alpha1",
		Resource: "activescenarios",
	}

	var scenarioID string

	// Check if active scenario exists
	active, err := m.dynamicClient.Resource(activeGVR).
		Get(context.TODO(), "current-playground-scenario", metav1.GetOptions{})
	if err != nil {
		// No CRD, but try to list Helm releases to find any devopsbeerer-* releases
		fmt.Printf("⚠️  No active scenario CRD found, checking for Helm releases...\n")
		cmd := exec.Command("helm", "list", "-A", "-o", "json")
		if _, err := cmd.Output(); err == nil {
			// Parse output to find devopsbeerer-* releases
			// For now, return error as we don't know which scenario to uninstall
			return fmt.Errorf("no active scenario found")
		}
		return fmt.Errorf("no active scenario found")
	} else {
		scenarioID, _, _ = unstructured.NestedString(active.Object, "spec", "scenarioId")
		fmt.Printf("🗑️  Uninstalling scenario: %s\n", scenarioID)
	}

	// Uninstall Helm release
	helmReleaseName := getHelmReleaseName(scenarioID)
	helmNamespace := getHelmNamespace(scenarioID)

	fmt.Printf("📦 Uninstalling Helm release: %s from namespace: %s\n", helmReleaseName, helmNamespace)
	cmd := exec.Command("helm", "uninstall", helmReleaseName, "-n", helmNamespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("⚠️  Helm uninstall warning: %s\n", string(output))
	} else {
		fmt.Printf("✅ Helm release uninstalled\n")
	}

	// Delete the ActiveScenario CRD
	if active != nil {
		err = m.dynamicClient.Resource(activeGVR).
			Delete(context.TODO(), "current-playground-scenario", metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete active scenario: %w", err)
		}
		fmt.Printf("✅ ActiveScenario resource deleted\n")
	}

	// Delete namespace
	fmt.Printf("📁 Ensuring namespace removed: %s\n", helmNamespace)
	cmd = exec.Command("kubectl", "delete", "namespace", helmNamespace)
	cmd.Run()

	return nil
}

// downloadChart downloads the Helm chart from GitHub
func (m *Manager) downloadChart(scenario *Scenario) (string, error) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "devopsbeerer-chart-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Clone the repository
	repoURL := scenario.HelmChart.Link
	if repoURL == "" {
		repoURL = "https://github.com/DevOpsBeerer/playground-scenarios-charts.git"
	}

	fmt.Printf("📂 Cloning repository: %s\n", repoURL)
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone repository: %w\n%s", err, string(output))
	}

	// Determine chart directory
	chartDir := scenario.HelmChart.Dir
	if chartDir == "" {
		chartDir = scenario.ID
	}

	fullChartPath := filepath.Join(tempDir, chartDir)
	if _, err := os.Stat(fullChartPath); os.IsNotExist(err) {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("chart directory '%s' not found in repository", chartDir)
	}

	return fullChartPath, nil
}

// installChart installs the downloaded Helm chart
func (m *Manager) installChart(chartPath string, scenarioID string) error {
	helmReleaseName := getHelmReleaseName(scenarioID)
	helmNamespace := getHelmNamespace(scenarioID)

	// Create namespace if it doesn't exist
	fmt.Printf("📁 Ensuring namespace exists: %s\n", helmNamespace)
	cmd := exec.Command("kubectl", "create", "namespace", helmNamespace, "--dry-run=client", "-o", "yaml")
	output, _ := cmd.Output()

	cmd = exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(string(output))
	cmd.Run()

	// Install Helm chart
	helmCmd := []string{
		"helm", "upgrade", "--install",
		helmReleaseName,
		chartPath,
		"-n", helmNamespace,
		"--create-namespace",
		"--wait",
		"--timeout", "5m",
		"--set", fmt.Sprintf("scenario.id=%s", scenarioID),
	}

	fmt.Printf("🚀 Running: %s\n", strings.Join(helmCmd, " "))
	cmd = exec.Command(helmCmd[0], helmCmd[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("helm install failed: %w\n%s", err, string(output))
	}

	fmt.Printf("✅ Helm chart installed successfully\n")
	return nil
}

// updateActiveScenarioStatus updates the status of the ActiveScenario
func (m *Manager) UpdateActiveScenarioStatus(scenarioID string, phase string, helmRelease string) error {
	// Define GVR for ActiveScenario
	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.io",
		Version:  "v1alpha1",
		Resource: "activescenarios",
	}

	// Get the singleton active scenario
	current, err := m.dynamicClient.Resource(activeGVR).
		Get(context.TODO(), "current-playground-scenario", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Update status
	status := map[string]interface{}{
		"phase":              phase,
		"helmReleaseName":    helmRelease,
		"startTime":          time.Now().Format(time.RFC3339),
		"lastTransitionTime": time.Now().Format(time.RFC3339),
	}

	// Get scenario name
	if scenario, err := m.GetScenario(scenarioID); err == nil {
		status["scenarioName"] = scenario.Name
	}

	if err := unstructured.SetNestedMap(current.Object, status, "status"); err != nil {
		return err
	}

	// Update the resource
	_, err = m.dynamicClient.Resource(activeGVR).
		UpdateStatus(context.TODO(), current, metav1.UpdateOptions{})
	return err
}

// showScenarioInfo displays information about the deployed scenario
func (m *Manager) showScenarioInfo(scenario *Scenario) {
	helmReleaseName := getHelmReleaseName(scenario.ID)
	helmNamespace := getHelmNamespace(scenario.ID)

	fmt.Println("\n📋 Scenario Information:")
	fmt.Println("------------------------")
	fmt.Printf("Name: %s\n", scenario.Name)
	fmt.Printf("ID: %s\n", scenario.ID)
	fmt.Printf("Description: %s\n", scenario.Description)

	if len(scenario.Features) > 0 {
		fmt.Println("\n🎯 Features:")
		for _, feature := range scenario.Features {
			fmt.Printf("  • %s\n", feature)
		}
	}

	fmt.Printf("\n⚙️  Helm Release: %s\n", helmReleaseName)
	fmt.Printf("📁 Namespace: %s\n", helmNamespace)

	fmt.Println("\n💡 Tips:")
	fmt.Printf("  • Check pods: kubectl get pods -n %s\n", helmNamespace)
	fmt.Printf("  • Check services: kubectl get svc -n %s\n", helmNamespace)
	fmt.Printf("  • Check ingress: kubectl get ingress -n %s\n", helmNamespace)
	fmt.Printf("  • View logs: kubectl logs -n %s <pod>\n", helmNamespace)
	fmt.Println("  • Check status: kubectl get activescenario")
	fmt.Printf("  • Get Helm values: helm get values %s -n %s\n", helmReleaseName, helmNamespace)
}

// GetScenarioStatus checks if a scenario is currently deployed
func (m *Manager) GetScenarioStatus() (*ScenarioStatus, error) {
	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.io",
		Version:  "v1alpha1",
		Resource: "activescenarios",
	}

	obj, err := m.dynamicClient.Resource(activeGVR).
		Get(context.TODO(), "current-playground-scenario", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("no active scenario found")
	}

	status := &ScenarioStatus{}

	// Extract spec info
	if scenarioID, found, _ := unstructured.NestedString(obj.Object, "spec", "scenarioId"); found {
		status.ScenarioID = scenarioID
	}

	// Extract status info
	if phase, found, _ := unstructured.NestedString(obj.Object, "status", "phase"); found {
		status.Phase = phase
	}
	if message, found, _ := unstructured.NestedString(obj.Object, "status", "message"); found {
		status.Message = message
	}
	if helmRelease, found, _ := unstructured.NestedString(obj.Object, "status", "helmReleaseName"); found {
		status.HelmRelease = helmRelease
	}
	if startTime, found, _ := unstructured.NestedString(obj.Object, "status", "startTime"); found {
		status.StartTime = startTime
	}

	// Also check Helm status
	if status.ScenarioID != "" {
		helmReleaseName := getHelmReleaseName(status.ScenarioID)
		helmNamespace := getHelmNamespace(status.ScenarioID)

		cmd := exec.Command("helm", "status", helmReleaseName, "-n", helmNamespace, "-o", "json")
		if _, err := cmd.Output(); err == nil {
			// Parse helm status if needed
			status.HelmStatus = "deployed"
		}
	}

	return status, nil
}

// ListScenarios fetches and returns all available scenarios from Kubernetes
func (m *Manager) ListScenarios() ([]Scenario, error) {
	// List all ScenarioDefinitions (cluster-scoped)
	list, err := m.dynamicClient.Resource(m.gvr).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list scenario definitions: %w", err)
	}

	scenarios := make([]Scenario, 0, len(list.Items))

	for _, item := range list.Items {
		scenario, err := m.unstructuredToScenario(&item)
		if err != nil {
			// Log error but continue with other scenarios
			fmt.Printf("Warning: failed to parse scenario %s: %v\n",
				item.GetName(), err)
			continue
		}
		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// unstructuredToScenario converts an unstructured object to Scenario
func (m *Manager) unstructuredToScenario(obj *unstructured.Unstructured) (Scenario, error) {
	var scenario Scenario

	// Extract spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return scenario, fmt.Errorf("spec not found")
	}

	// Extract fields from spec
	if name, ok := spec["name"].(string); ok {
		scenario.Name = name
	}
	if id, ok := spec["id"].(string); ok {
		scenario.ID = id
	}
	if desc, ok := spec["description"].(string); ok {
		scenario.Description = desc
	}

	// Extract tags
	if tags, found, err := unstructured.NestedStringSlice(obj.Object, "spec", "tags"); err == nil && found {
		scenario.Tags = tags
	}

	// Extract features
	if features, found, err := unstructured.NestedStringSlice(obj.Object, "spec", "features"); err == nil && found {
		scenario.Features = features
	}

	// Extract helmChart
	if helmChart, found, err := unstructured.NestedMap(obj.Object, "spec", "helmChart"); err == nil && found {
		if link, ok := helmChart["link"].(string); ok {
			scenario.HelmChart.Link = link
		}
		if dir, ok := helmChart["dir"].(string); ok {
			scenario.HelmChart.Dir = dir
		}
	}

	return scenario, nil
}

// GetScenario fetches a specific scenario by ID
func (m *Manager) GetScenario(id string) (*Scenario, error) {
	// List all scenarios and find by ID
	scenarios, err := m.ListScenarios()
	if err != nil {
		return nil, err
	}

	for _, scenario := range scenarios {
		if scenario.ID == id {
			return &scenario, nil
		}
	}

	return nil, fmt.Errorf("scenario with ID '%s' not found", id)
}

// FindScenario finds a scenario by ID
func (m *Manager) FindScenario(id string) (*Scenario, error) {
	scenarios, err := m.ListScenarios()
	if err != nil {
		return nil, err
	}

	for _, scenario := range scenarios {
		if scenario.ID == id {
			return &scenario, nil
		}
	}

	return nil, fmt.Errorf("scenario '%s' not found", id)
}

func (m *Manager) GetActiveScenario() (*ActiveScenarioInfo, error) {
	// Define GVR for ActiveScenario
	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.io",
		Version:  "v1alpha1",
		Resource: "activescenarios",
	}

	// Get the singleton active scenario
	obj, err := m.dynamicClient.Resource(activeGVR).
		Get(context.TODO(), "current-playground-scenario", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("no active scenario found: %w", err)
	}

	info := &ActiveScenarioInfo{}

	// Extract scenario ID
	if scenarioID, found, err := unstructured.NestedString(obj.Object, "spec", "scenarioId"); err == nil && found {
		info.ScenarioID = scenarioID
	}

	// Extract status phase
	if phase, found, err := unstructured.NestedString(obj.Object, "status", "phase"); err == nil && found {
		info.Phase = phase
	}

	// Extract scenario name from status
	if name, found, err := unstructured.NestedString(obj.Object, "status", "scenarioName"); err == nil && found {
		info.ScenarioName = name
	}

	return info, nil
}
