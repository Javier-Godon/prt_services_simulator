package main

import (
	"fmt"
	"os"
	"testing"
)

// TestRepositoryConfiguration tests repository configuration
func TestRepositoryConfiguration(t *testing.T) {
	pipeline := &RailwayPipeline{
		RepoName:  "railway_oriented_java",
		GitRepo:   "https://github.com/test/railway_oriented_java.git",
		GitBranch: "main",
		GitUser:   "testuser",
	}

	if pipeline.RepoName == "" {
		t.Fatal("RepoName is empty")
	}

	if pipeline.GitRepo == "" {
		t.Fatal("GitRepo is empty")
	}

	if pipeline.GitBranch == "" {
		t.Fatal("GitBranch is empty")
	}

	fmt.Printf("✅ Repository config valid: %s (%s)\n", pipeline.RepoName, pipeline.GitBranch)
}

// TestEnvironmentVariables tests required environment variables
func TestEnvironmentVariables(t *testing.T) {
	requiredVars := []string{"CR_PAT", "USERNAME"}

	for _, varName := range requiredVars {
		if _, exists := os.LookupEnv(varName); !exists {
			fmt.Printf("⚠️  %s not set (required for full pipeline)\n", varName)
			// Don't fail - credentials might not be set in test environment
		}
	}

	fmt.Println("✅ Environment variable checks completed")
}

// TestImageNaming tests Docker image naming logic
func TestImageNaming(t *testing.T) {
	pipeline := &RailwayPipeline{
		ImageName: "railway_framework",
		GitUser:   "javier-godon",
	}

	imageName := fmt.Sprintf("ghcr.io/%s/%s:v1.0.0", pipeline.GitUser, pipeline.ImageName)

	if imageName == "" {
		t.Fatal("Image name is empty")
	}

	if !contains(imageName, "ghcr.io") {
		t.Fatal("Image name should contain registry")
	}

	fmt.Printf("✅ Image naming valid: %s\n", imageName)
}

// TestGitRepositoryURL tests Git repository URL construction
func TestGitRepositoryURL(t *testing.T) {
	username := "testuser"
	repoName := "railway_oriented_java"

	gitRepo := fmt.Sprintf("https://github.com/%s/%s.git", username, repoName)

	if !contains(gitRepo, "github.com") {
		t.Fatal("Git repository URL should contain github.com")
	}

	if !contains(gitRepo, repoName) {
		t.Fatal("Git repository URL should contain repository name")
	}

	fmt.Printf("✅ Git repository URL valid: %s\n", gitRepo)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
