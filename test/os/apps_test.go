package os_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMS05_Apps_FilesExist verifies that all files for Milestone MS-OS-05
// (App manifest schema, templates, and documentation) are present in the repository.
func TestMS05_Apps_FilesExist(t *testing.T) {
	requiredFiles := []string{
		"apps/schema/app-manifest.json",
		"apps/templates/zigbee2mqtt/domus-app.yaml",
		"apps/templates/mosquitto/domus-app.yaml",
		"docs/os/MS-OS-05-APPS-ENGINE-AND-SANDBOXING.md",
	}

	baseDir := findRepoRoot(t)

	for _, relPath := range requiredFiles {
		fullPath := filepath.Join(baseDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected app engine file missing: %s (err: %v)", relPath, err)
			continue
		}
		if info.IsDir() {
			t.Errorf("expected file but got directory: %s", relPath)
		}
		if info.Size() == 0 {
			t.Errorf("file is empty: %s", relPath)
		}
	}
}

// TestMS05_Schema_Validity ensures apps/schema/app-manifest.json is valid JSON
// and contains required schema validation fields.
func TestMS05_Schema_Validity(t *testing.T) {
	baseDir := findRepoRoot(t)
	schemaPath := filepath.Join(baseDir, "apps", "schema", "app-manifest.json")

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read app-manifest.json: %v", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("app-manifest.json is not valid JSON: %v", err)
	}

	if schema["$schema"] == nil {
		t.Errorf("missing $schema declaration")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing properties map in schema")
	}

	requiredFields := []string{
		"id",
		"name",
		"version",
		"image",
		"permissions",
		"resources",
	}

	for _, rf := range requiredFields {
		if properties[rf] == nil {
			t.Errorf("schema properties missing definition for: %s", rf)
		}
	}
}

// TestMS05_Zigbee2MQTT_Template_Properties verifies the reference Zigbee2MQTT template
// contains proper hardware isolation and MQTT permissions.
func TestMS05_Zigbee2MQTT_Template_Properties(t *testing.T) {
	baseDir := findRepoRoot(t)
	tplPath := filepath.Join(baseDir, "apps", "templates", "zigbee2mqtt", "domus-app.yaml")

	data, err := os.ReadFile(tplPath)
	if err != nil {
		t.Fatalf("failed to read zigbee2mqtt domus-app.yaml: %v", err)
	}

	content := string(data)

	requiredTokens := []string{
		"id: zigbee2mqtt",
		"name: Zigbee2MQTT",
		"/dev/ttyUSB0",
		"zigbee2mqtt/#",
		"memory_limit_mb: 256",
		"bridge",
	}

	for _, token := range requiredTokens {
		if !strings.Contains(content, token) {
			t.Errorf("zigbee2mqtt template missing token: %s", token)
		}
	}
}

// TestMS05_Mosquitto_Template_Properties verifies the reference Mosquitto template.
func TestMS05_Mosquitto_Template_Properties(t *testing.T) {
	baseDir := findRepoRoot(t)
	tplPath := filepath.Join(baseDir, "apps", "templates", "mosquitto", "domus-app.yaml")

	data, err := os.ReadFile(tplPath)
	if err != nil {
		t.Fatalf("failed to read mosquitto domus-app.yaml: %v", err)
	}

	content := string(data)

	requiredTokens := []string{
		"id: mosquitto",
		"1883",
		"memory_limit_mb: 128",
	}

	for _, token := range requiredTokens {
		if !strings.Contains(content, token) {
			t.Errorf("mosquitto template missing token: %s", token)
		}
	}
}
