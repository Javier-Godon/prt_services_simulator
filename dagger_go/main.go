package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"dagger.io/dagger"
)

// Constants for Maven and build configuration
const (
	baseImage            = "amazoncorretto:25.0.1"
	appWorkdir           = "/app/prt_services_simulator"
	mavenReleaseVersion  = "25"
	mavenCompilerPreview = "--enable-preview"
	mavenCompilerRelease = "-Dmaven.compiler.release="
	mavenCompilerArgs    = "-Dmaven.compiler.compilerArgs="
	separatorLine        = "─────────────────────────────────────────────────────────────────────────────────"
)

// SimulatorPipeline represents the PRT Services Simulator CI/CD pipeline
type SimulatorPipeline struct {
	RepoName        string
	ImageName       string
	GitRepo         string
	GitBranch       string
	GitUser         string
	GitHost         string // e.g. github.com, gitlab.com, bitbucket.org
	GitAuthUsername string // HTTP auth username for cloning (x-access-token for GitHub PAT, oauth2 for GitLab)
	Registry        string // container registry host, e.g. ghcr.io, docker.io, registry.gitlab.com
	RegistryUser    string // registry namespace / org (defaults to GitUser)
	MavenCache      *dagger.CacheVolume
}

// main runs the PRT Services Simulator CI/CD pipeline
func main() {
	ctx := context.Background()

	// Check for required environment variables
	requiredVars := []string{"CR_PAT", "USERNAME"}
	for _, varName := range requiredVars {
		if _, exists := os.LookupEnv(varName); !exists {
			fmt.Fprintf(os.Stderr, "ERROR: %s environment variable must be set\n", varName)
			os.Exit(1)
		}
	}

	// Get repository information from environment
	repoName := os.Getenv("REPO_NAME")
	if repoName == "" {
		repoName = "prt_services_simulator"
	}

	gitHost := os.Getenv("GIT_HOST")
	if gitHost == "" {
		gitHost = "github.com"
	}

	gitAuthUsername := os.Getenv("GIT_AUTH_USERNAME")
	if gitAuthUsername == "" {
		gitAuthUsername = "x-access-token" // default for GitHub PAT; use "oauth2" for GitLab
	}

	registry := os.Getenv("REGISTRY")
	if registry == "" {
		registry = "ghcr.io"
	}

	username := os.Getenv("USERNAME")

	registryUser := os.Getenv("REGISTRY_USERNAME")
	if registryUser == "" {
		registryUser = username
	}

	gitRepo := os.Getenv("GIT_REPO")
	if gitRepo == "" {
		gitRepo = fmt.Sprintf("https://%s/%s/%s.git", gitHost, username, repoName)
	}

	gitBranch := os.Getenv("GIT_BRANCH")
	if gitBranch == "" {
		gitBranch = "main"
	}

	imageNameEnv := os.Getenv("IMAGE_NAME")
	if imageNameEnv == "" {
		imageNameEnv = repoName
	}

	fmt.Printf("🚀 Starting %s CI/CD Pipeline (Go SDK v0.19.7)...\n", repoName)
	fmt.Printf("   Repository: %s (branch: %s)\n", gitRepo, gitBranch)
	fmt.Printf("   Registry:   %s/%s\n", registry, strings.ToLower(registryUser))

	// Initialize Dagger client
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create Dagger client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	pipeline := &SimulatorPipeline{
		RepoName:        repoName,
		ImageName:       imageNameEnv,
		GitRepo:         gitRepo,
		GitBranch:       gitBranch,
		GitUser:         username,
		GitHost:         gitHost,
		GitAuthUsername: gitAuthUsername,
		Registry:        registry,
		RegistryUser:    registryUser,
	}

	// Run pipeline stages
	if err := pipeline.run(ctx, client); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Pipeline failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n🎉 Pipeline completed successfully!")
}

// run executes the complete CI/CD pipeline:
// Clone → Test → Build JAR → Docker Build → Publish to GHCR
func (p *SimulatorPipeline) run(ctx context.Context, client *dagger.Client) error {
	// ── Clone repository ──────────────────────────────────────────
	fmt.Printf("📥 Cloning repository: %s (branch: %s)\n", p.GitRepo, p.GitBranch)

	gitURL := fmt.Sprintf("https://%s/%s/%s.git", p.GitHost, p.GitUser, p.RepoName)
	crPAT := client.SetSecret("git-pat", os.Getenv("CR_PAT"))

	repo := client.Git(gitURL, dagger.GitOpts{
		KeepGitDir:       true,
		HTTPAuthToken:    crPAT,
		HTTPAuthUsername: p.GitAuthUsername,
	})

	source := repo.Branch(p.GitBranch).Tree()

	commitSHA, err := repo.Branch(p.GitBranch).Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get commit SHA: %w", err)
	}
	latestCommit := commitSHA
	fmt.Printf("   Commit: %s\n", latestCommit[:min(12, len(latestCommit))])

	// ── Set up build environment ─────────────────────────────────
	fmt.Println("🔨 Setting up build environment...")

	p.MavenCache = client.CacheVolume("maven-cache-simulator")

	builder := client.Container().
		From(baseImage).
		WithExec([]string{"yum", "install", "-y", "maven"}).
		WithMountedCache("/root/.m2", p.MavenCache).
		WithMountedDirectory(appWorkdir, source).
		WithWorkdir(appWorkdir)

	// ── Stage 1: Test ────────────────────────────────────────────
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PIPELINE STAGE 1: TEST EXECUTION")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("🧪 Running all tests (Spring Boot MockMvc)...")
	fmt.Println("🏃 Executing: mvn test")
	fmt.Println(separatorLine)

	testContainer := builder.WithExec([]string{
		"mvn", "test",
		mavenCompilerRelease + mavenReleaseVersion,
		mavenCompilerArgs + mavenCompilerPreview,
	})

	_, err = testContainer.Stdout(ctx)
	if err != nil {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("❌ PIPELINE FAILED AT STAGE 1: TEST EXECUTION")
		fmt.Println(strings.Repeat("=", 80))
		return fmt.Errorf("tests failed - aborting build: %w", err)
	}
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ STAGE 1 COMPLETE: All tests passed")
	fmt.Println(strings.Repeat("=", 80))

	// ── Stage 2: Build JAR ───────────────────────────────────────
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PIPELINE STAGE 2: BUILD ARTIFACT")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("📦 Building Maven artifact (JAR file)...")
	fmt.Println("🏃 Executing: mvn package -DskipTests")

	buildContainer := testContainer.WithExec([]string{
		"mvn", "package",
		"-DskipTests",
		mavenCompilerRelease + mavenReleaseVersion,
		mavenCompilerArgs + mavenCompilerPreview,
		"-q",
	})

	_, err = buildContainer.Stdout(ctx)
	if err != nil {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("❌ PIPELINE FAILED AT STAGE 2: BUILD ARTIFACT")
		fmt.Println(strings.Repeat("=", 80))
		return fmt.Errorf("failed to build JAR: %w", err)
	}
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ STAGE 2 COMPLETE: Build successful")
	fmt.Println(strings.Repeat("=", 80))

	// ── Stage 3: Build Docker image ──────────────────────────────
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PIPELINE STAGE 3: BUILD DOCKER IMAGE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("🐳 Building Docker image...")

	// Get the project directory for Docker build (Dockerfile is at the root)
	projectDir := buildContainer.Directory(appWorkdir)
	image := projectDir.DockerBuild()

	// Create image tags
	shortSHA := latestCommit
	if len(latestCommit) > 7 {
		shortSHA = latestCommit[:7]
	}
	timestamp := time.Now().Format("20060102-1504")
	imageTag := fmt.Sprintf("v0.1.0-%s-%s", shortSHA, timestamp)

	// Docker-safe naming
	imageNameClean := strings.ToLower(strings.ReplaceAll(p.ImageName, "_", "-"))
	registryUser := strings.ToLower(p.RegistryUser)
	imageName := fmt.Sprintf("%s/%s/%s:%s", p.Registry, registryUser, imageNameClean, imageTag)
	latestImageName := fmt.Sprintf("%s/%s/%s:latest", p.Registry, registryUser, imageNameClean)

	// ── Stage 4: Publish to registry ─────────────────────────────
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("PIPELINE STAGE 4: PUBLISH TO %s\n", strings.ToUpper(p.Registry))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("📤 Publishing to: %s\n", imageName)

	password := client.SetSecret("password", os.Getenv("CR_PAT"))
	publishedAddress, err := image.
		WithRegistryAuth(p.Registry, p.RegistryUser, password).
		Publish(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to publish versioned image: %w", err)
	}

	latestAddress, err := image.
		WithRegistryAuth(p.Registry, p.RegistryUser, password).
		Publish(ctx, latestImageName)
	if err != nil {
		return fmt.Errorf("failed to publish latest image: %w", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ STAGE 4 COMPLETE: Images published")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("   📦 Versioned: %s\n", publishedAddress)
	fmt.Printf("   📦 Latest:    %s\n", latestAddress)

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
