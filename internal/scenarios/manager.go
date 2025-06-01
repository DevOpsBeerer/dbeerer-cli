package scenarios

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
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
		Group:    "devopsbeerer.ch",
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

	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.ch",
		Version:  "v1alpha1",
		Resource: "activescenarios",
	}

	// Try to get existing active scenario
	_ = m.dynamicClient.Resource(activeGVR).Delete(context.TODO(), "current-playground-scenario", metav1.DeleteOptions{})

	// Create the ActiveScenario CRD first
	fmt.Printf("📝 Creating ActiveScenario resource...\n")

	activeScenario := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "devopsbeerer.ch/v1alpha1",
			"kind":       "ActiveScenario",
			"metadata": map[string]any{
				"name": "current-playground-scenario",
			},
			"spec": map[string]any{
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

	fmt.Printf("🔁 Scenario '%s' is getting installed\n", scenario.Name)

	return nil
}

// UninstallScenario removes the current scenario deployment
func (m *Manager) UninstallScenario() error {
	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.ch",
		Version:  "v1alpha1",
		Resource: "activescenarios",
	}

	err := m.dynamicClient.Resource(activeGVR).Delete(context.TODO(), "current-playground-scenario", metav1.DeleteOptions{})

	if err != nil {
		fmt.Printf("No active scenario found")
		return err
	}

	fmt.Printf("✅ Active scenario is getting deleted")

	return nil
}

// updateActiveScenarioStatus updates the status of the ActiveScenario
func (m *Manager) UpdateActiveScenarioStatus(scenarioID string, phase string, helmRelease string) error {
	// Define GVR for ActiveScenario
	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.ch",
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

// GetScenarioStatus checks if a scenario is currently deployed
func (m *Manager) GetScenarioStatus() (*ScenarioStatus, error) {
	activeGVR := schema.GroupVersionResource{
		Group:    "devopsbeerer.ch",
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
		Group:    "devopsbeerer.ch",
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
