package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"dagger.io/dagger"
)

// Constants for Maven and build configuration
const (
	baseImage                 = "amazoncorretto:25.0.1"
	appWorkdir                = "/app/railway_framework"
	containerDockerSocketPath = "/var/run/docker.sock" // Path inside container (always Linux)
	dockerUnixPrefix          = "unix://"
	mavenReleaseVersion       = "25"
	mavenCompilerPreviewFlag  = "--enable-preview"
	mavenCompilerRelease      = "-Dmaven.compiler.release="
	mavenCompilerArgs         = "-Dmaven.compiler.compilerArgs="
	separatorLine             = "─────────────────────────────────────────────────────────────────────────────────"
)

// RailwayPipeline represents the Railway-Oriented Java framework CI/CD pipeline
type RailwayPipeline struct {
	RepoName            string
	ImageName           string
	GitRepo             string
	GitBranch           string
	GitUser             string
	MavenCache          *dagger.CacheVolume
	ContainerImg        *dagger.Container
	HasDocker           bool // Docker availability for testcontainers
	RunUnitTests        bool // Whether to run unit tests
	RunIntegrationTests bool // Whether to run integration tests
}

// getDockerSocketPath returns the Docker socket path for the current platform
// Returns empty string if Docker is not available
func getDockerSocketPath() string {
	var candidates []string

	switch runtime.GOOS {
	case "windows":
		// Windows: Docker Desktop uses named pipe
		candidates = []string{
			`\\.\pipe\docker_engine`, // Docker Desktop for Windows
			`//./pipe/docker_engine`, // Alternative format
		}
	case "darwin":
		// macOS: Docker Desktop socket location
		candidates = []string{
			"/var/run/docker.sock",                         // Standard location
			os.Getenv("HOME") + "/.docker/run/docker.sock", // Docker Desktop
			os.Getenv("HOME") + "/.colima/docker.sock",     // Colima
		}
	default:
		// Linux and others: standard Unix socket
		candidates = []string{
			"/var/run/docker.sock",   // Standard location
			"/run/docker.sock",       // Alternative location
			os.Getenv("DOCKER_HOST"), // Explicit DOCKER_HOST env var
		}
	}

	// Try each candidate path
	for _, path := range candidates {
		if path == "" {
			continue
		}
		// Remove unix:// prefix if present (from DOCKER_HOST)
		path = strings.TrimPrefix(path, dockerUnixPrefix)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// main runs the Railway-Oriented Programming framework CI/CD pipeline in Go
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
		repoName = "railway_oriented_java"
	}

	gitRepo := os.Getenv("GIT_REPO")
	if gitRepo == "" {
		username := os.Getenv("USERNAME")
		gitRepo = fmt.Sprintf("https://github.com/%s/%s.git", username, repoName)
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

	// Parse test execution flags from environment variables
	runUnitTests := parseEnvBool("RUN_UNIT_TESTS", true)               // Default: true
	runIntegrationTests := parseEnvBool("RUN_INTEGRATION_TESTS", true) // Default: true

	fmt.Printf("🧪 Test Configuration:\n")
	fmt.Printf("   Unit tests: %v (override with RUN_UNIT_TESTS=false)\n", runUnitTests)
	fmt.Printf("   Integration tests: %v (override with RUN_INTEGRATION_TESTS=false)\n", runIntegrationTests)

	if !runUnitTests && !runIntegrationTests {
		fmt.Fprintf(os.Stderr, "ERROR: At least one of RUN_UNIT_TESTS or RUN_INTEGRATION_TESTS must be true\n")
		os.Exit(1)
	}

	// Initialize Dagger client
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create Dagger client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	pipeline := &RailwayPipeline{
		RepoName:            repoName,
		ImageName:           imageNameEnv,
		GitRepo:             gitRepo,
		GitBranch:           gitBranch,
		GitUser:             os.Getenv("USERNAME"),
		RunUnitTests:        runUnitTests,
		RunIntegrationTests: runIntegrationTests,
	}

	// Run pipeline stages
	if err := pipeline.run(ctx, client); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Pipeline failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n🎉 Pipeline completed successfully!")
}

// run executes the complete CI/CD pipeline with the correct order:
// Clone → Test (Unit + Integration with Docker) → Build → Dockerize → Publish
func (p *RailwayPipeline) run(ctx context.Context, client *dagger.Client) error {
	// NOTE: Maven cache volume is created in setupBuilder() with a key based on test mode
	// DO NOT create it here with a fixed key - that would bypass the test-mode-specific caching

	// Clone repository from GitHub
	fmt.Printf("📥 Cloning repository: %s (branch: %s)\n", p.GitRepo, p.GitBranch)

	// Set up build environment with cloned source
	fmt.Println("🔨 Setting up build environment...")

	// Clone the repository using Dagger's native git support with authentication
	fmt.Println("🔖 Getting Git repository...")

	// Use simple HTTPS URL with HTTPAuthToken option (documented, official way)
	gitURL := fmt.Sprintf("https://github.com/%s/%s.git", p.GitUser, p.RepoName)
	crPAT := client.SetSecret("github-pat", os.Getenv("CR_PAT"))

	repo := client.Git(gitURL, dagger.GitOpts{
		KeepGitDir:       true,
		HTTPAuthToken:    crPAT,            // Use documented HTTPAuthToken field
		HTTPAuthUsername: "x-access-token", // GitHub's convention for PAT
	})

	// Get the source at the specified branch
	source := repo.Branch(p.GitBranch).Tree()

	// Get commit SHA
	commitSHA, err := repo.Branch(p.GitBranch).Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get commit SHA: %w", err)
	}
	latestCommit := commitSHA
	fmt.Printf("   Commit: %s\n", latestCommit[:min(12, len(latestCommit))])

	// Check if Docker socket is available on the host BEFORE trying to mount
	fmt.Println("🔍 Checking Docker availability for testcontainers...")
	var hasDocker bool
	var hostDockerSocketPath string

	// Detect Docker socket path for current platform
	hostDockerSocketPath = getDockerSocketPath()
	if hostDockerSocketPath != "" {
		hasDocker = true
		fmt.Printf("✅ Docker socket detected on host: %s\n", hostDockerSocketPath)
		fmt.Println("   Mounting for full test suite (unit + integration)")
	} else {
		hasDocker = false
		fmt.Printf("⚠️  Docker socket NOT available on host (OS: %s)\n", runtime.GOOS)
		fmt.Println("   Will run unit tests only")
	}
	p.HasDocker = hasDocker

	// Determine if we need Docker for this run
	needsDocker := p.RunIntegrationTests && hasDocker

	// Set up builder with Docker socket mounting only if integration tests are enabled and Docker is available
	builder := p.setupBuilder(ctx, client, baseImage, source, needsDocker, hostDockerSocketPath)

	// Stage 1: Run tests FIRST (before build)
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PIPELINE STAGE 1: TEST EXECUTION")
	fmt.Println(strings.Repeat("=", 80))
	testContainer, testErr := p.runTests(ctx, client, builder, source, hasDocker)
	if testErr != nil {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("❌ PIPELINE FAILED AT STAGE 1: TEST EXECUTION")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("Error: %v\n", testErr)
		return fmt.Errorf("tests failed - aborting build: %w", testErr)
	}
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ STAGE 1 COMPLETE: All tests passed")
	fmt.Println(strings.Repeat("=", 80))

	// Use testContainer for subsequent stages (may include test outputs)
	builder = testContainer

	// Stage 2: Build JAR (only after successful tests)
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PIPELINE STAGE 2: BUILD ARTIFACT")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("📦 Building Maven artifact (JAR file)...")
	fmt.Println("🏃 Executing: mvn package -DskipTests")
	fmt.Println("")

	buildContainer := builder.WithExec([]string{
		"mvn", "package",
		"-DskipTests", // Tests already ran
		mavenCompilerRelease + mavenReleaseVersion,
		mavenCompilerArgs + mavenCompilerPreviewFlag,
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

	// Stage 3: Build Docker image from the built JAR
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PIPELINE STAGE 3: BUILD DOCKER IMAGE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("🐳 Building Docker image...")

	// Get railway_framework directory for Docker build (Dockerfile is here)
	railwayFrameworkDir := buildContainer.WithWorkdir(appWorkdir).Directory(appWorkdir)
	image := railwayFrameworkDir.DockerBuild()

	// Create image tags
	shortSHA := latestCommit
	if len(latestCommit) > 7 {
		shortSHA = latestCommit[:7]
	}
	timestamp := time.Now().Format("20060102-1504")
	imageTag := fmt.Sprintf("v1.0.0-%s-%s", shortSHA, timestamp)

	// Docker-safe naming
	imageNameClean := strings.ToLower(strings.ReplaceAll(p.ImageName, "_", "-"))
	username := p.GitUser
	imageName := fmt.Sprintf("ghcr.io/%s/%s:%s", strings.ToLower(username), imageNameClean, imageTag)
	latestImageName := fmt.Sprintf("ghcr.io/%s/%s:latest", strings.ToLower(username), imageNameClean)

	// Stage 4: Publish images to GitHub Container Registry
	fmt.Printf("📤 Publishing to: %s\n", imageName)

	password := client.SetSecret("password", os.Getenv("CR_PAT"))
	publishedAddress, err := image.
		WithRegistryAuth("ghcr.io", username, password).
		Publish(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to publish versioned image: %w", err)
	}

	// Publish latest tag
	latestAddress, err := image.
		WithRegistryAuth("ghcr.io", username, password).
		Publish(ctx, latestImageName)
	if err != nil {
		return fmt.Errorf("failed to publish latest image: %w", err)
	}

	// Print results
	fmt.Println("✅ Images published:")
	fmt.Printf("   📦 Versioned: %s\n", publishedAddress)
	fmt.Printf("   📦 Latest: %s\n", latestAddress)

	// Trigger deployment webhook if configured
	if deployWebhook := os.Getenv("DEPLOY_WEBHOOK"); deployWebhook != "" {
		fmt.Println("🚀 Triggering deployment webhook...")
		if err := p.triggerWebhook(deployWebhook, imageTag, publishedAddress, latestCommit, timestamp); err != nil {
			fmt.Printf("⚠️  Warning: Deployment trigger failed: %v\n", err)
		} else {
			fmt.Println("✅ Deployment triggered successfully")
		}
	}

	return nil
}

// triggerWebhook sends deployment webhook notification
func (p *RailwayPipeline) triggerWebhook(webhookURL, imageTag, imageAddress, commitSHA, timestamp string) error {
	// This would integrate with your deployment system
	// Example: using webhook to trigger ArgoCD, Flux, or custom deployment service
	fmt.Printf("   Webhook: %s\n", webhookURL)
	fmt.Printf("   Image Tag: %s\n", imageTag)
	fmt.Printf("   Image: %s\n", imageAddress)
	fmt.Printf("   Commit: %s\n", commitSHA)
	fmt.Printf("   Timestamp: %s\n", timestamp)
	return nil
}

// setupBuilder creates the build environment with Docker support for testcontainers
// This sets up proper mounting of Docker socket when available AND when integration tests are enabled
func (p *RailwayPipeline) setupBuilder(ctx context.Context, client *dagger.Client, baseImage string, source *dagger.Directory, needsDocker bool, hostDockerSocketPath string) *dagger.Container {
	// Create Maven cache volume with key based on test mode to avoid conflicts
	// CRITICAL: Different cache keys prevent old compiled test classes from persisting across runs
	// When running unit-only tests vs full suite, we need separate caches so Maven's file-level
	// exclusion patterns work correctly. Otherwise Surefire finds old compiled classfiles and runs them.
	if p.MavenCache == nil {
		cacheKey := "maven-cache"
		// Use different cache keys for different test modes to prevent cache pollution
		if p.RunIntegrationTests && p.RunUnitTests {
			cacheKey = "maven-cache-full-suite" // Full suite including integration tests
		} else if p.RunUnitTests {
			cacheKey = "maven-cache-unit-only" // Unit tests only (integration excluded)
		} else if p.RunIntegrationTests {
			cacheKey = "maven-cache-integration-only" // Integration tests only
		}
		p.MavenCache = client.CacheVolume(cacheKey)
	}

	// Base setup with Maven and Java tools
	builder := client.Container().
		From(baseImage).
		WithExec([]string{"yum", "install", "-y", "maven", "git", "docker"}).
		WithMountedCache("/root/.m2", p.MavenCache).
		WithMountedDirectory("/app", source).
		WithWorkdir(appWorkdir)

	// Mount Docker socket for testcontainers support when available AND when integration tests are needed
	// This enables integration tests that use testcontainers (e.g., PostgreSQL)
	if needsDocker && hostDockerSocketPath != "" {
		fmt.Printf("   🔗 Mounting Docker socket for testcontainers\n")
		// Mount the Docker socket from the host - WithUnixSocket enables actual socket communication
		// (WithMountedFile only creates a file reference without socket communication capabilities)
		// Always mount to the standard Linux path inside the container (containers are Linux-based)
		builder = builder.WithUnixSocket(containerDockerSocketPath, client.Host().UnixSocket(hostDockerSocketPath))

		// Set environment variables for Testcontainers to find and use the Docker socket
		// TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE tells Testcontainers where the socket is mounted inside container
		builder = builder.WithEnvVariable("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", containerDockerSocketPath)
		// DOCKER_HOST tells Docker CLI and other tools where to find Docker (unix:// format with 3 slashes)
		builder = builder.WithEnvVariable("DOCKER_HOST", dockerUnixPrefix+containerDockerSocketPath)

		// Disable Ryuk (resource reaper) for CI/CD environments
		// Ryuk requires network connectivity between containers which doesn't work in Dagger's network isolation
		// Testcontainers will still work but won't automatically clean up containers (Dagger cleans up anyway)
		builder = builder.WithEnvVariable("TESTCONTAINERS_RYUK_DISABLED", "true")

		// KNOWN LIMITATION: Testcontainers integration tests cannot run in Dagger pipeline on Linux
		// Root cause: Docker wormhole pattern requires source code mounted at SAME absolute path
		//   - Host: /home/javier/.../railway_framework
		//   - Dagger container: /app/railway_framework (DIFFERENT PATH)
		// Testcontainers uses this path for volume bindings - mismatch breaks networking
		// Auto-detection doesn't work because paths don't match
		// Solution: Run integration tests locally or in environments with native Docker access (not nested containers)
		//
		// For Dagger pipeline: Integration tests are DISABLED - only unit tests run
		// See: https://java.testcontainers.org/features/networking/#exposing-host-ports-to-the-container
	}

	return builder
}

// runTests orchestrates test execution:
// - Unit tests run inside Dagger container (fast, isolated)
// - Integration tests run on host machine (avoids Docker-in-Docker networking issues)
// Test execution matrix:
//
//	RUN_UNIT_TESTS=true, RUN_INTEGRATION_TESTS=true, Docker=available
//	  → Unit tests in container + Integration tests on host
//	RUN_UNIT_TESTS=true, RUN_INTEGRATION_TESTS=true, Docker=unavailable
//	  → Unit tests only (integration tests skipped)
//	RUN_UNIT_TESTS=true, RUN_INTEGRATION_TESTS=false, Docker=any
//	  → Unit tests only (integration tests explicitly disabled)
//	RUN_UNIT_TESTS=false, RUN_INTEGRATION_TESTS=true, Docker=available
//	  → Integration tests on host only
//	RUN_UNIT_TESTS=false, RUN_INTEGRATION_TESTS=true, Docker=unavailable
//	  → No tests run (integration tests require Docker)
func (p *RailwayPipeline) runTests(ctx context.Context, client *dagger.Client, builder *dagger.Container, source *dagger.Directory, hasDocker bool) (*dagger.Container, error) {
	// Determine what tests to run
	runUnit := p.RunUnitTests
	runIntegration := p.RunIntegrationTests && hasDocker // Integration tests require Docker

	if !runUnit && !runIntegration {
		fmt.Println("   ⏭️  Skipping all tests (integration tests require Docker or are disabled)")
		return builder, nil
	}

	var testContainer *dagger.Container = builder
	var err error

	// Execute unit tests inside Dagger container
	if runUnit {
		testContainer, err = p.runUnitTests(ctx, testContainer)
		if err != nil {
			return nil, fmt.Errorf("unit tests failed: %w", err)
		}
	}

	// Execute integration tests on host machine (outside Dagger container)
	// This avoids Docker-in-Docker networking issues with Testcontainers
	if runIntegration {
		err = p.runIntegrationTestsOnHost(ctx, client, source)
		if err != nil {
			return nil, fmt.Errorf("integration tests failed: %w", err)
		}
	}

	return testContainer, nil
}

// runUnitTests executes unit tests inside the Dagger container
// Unit tests are fast, don't require Docker, and run in isolation
func (p *RailwayPipeline) runUnitTests(ctx context.Context, builder *dagger.Container) (*dagger.Container, error) {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  STAGE: Unit Tests Execution (Dagger Container)                              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("📍 Location: Inside Dagger container (isolated environment)")
	fmt.Println("⚡ Characteristics: Fast, no external dependencies, pure business logic")
	fmt.Println("")
	fmt.Println("⚙️  Configuration:")
	fmt.Printf("   • Test Pattern: !*IntegrationTest (excludes integration tests)\n")
	fmt.Printf("   • Java Version: %s (with preview features)\n", mavenReleaseVersion)
	fmt.Printf("   • Expected Test Count: ~58 unit tests\n")
	fmt.Println("")
	fmt.Println("🏃 Executing: mvn test -Dtest=!*IntegrationTest")
	fmt.Println(separatorLine)

	testCmd := []string{
		"mvn", "test",
		// Use -Dtest pattern to run ALL tests EXCEPT those matching *IntegrationTest
		"-Dtest=!*IntegrationTest",
		mavenCompilerRelease + mavenReleaseVersion,
		mavenCompilerArgs + mavenCompilerPreviewFlag,
	}

	testContainer := builder.WithExec(testCmd)

	// Execute tests and capture output
	_, err := testContainer.Stdout(ctx)

	fmt.Println(separatorLine)

	if err != nil {
		fmt.Println("\n❌ FAILED: Unit tests failed")
		fmt.Println("   Check test output above for details")
		return nil, err
	}

	fmt.Println("\n✅ SUCCESS: All unit tests passed")
	fmt.Println("")

	return testContainer, nil
}

// runIntegrationTestsOnHost executes integration tests TRULY on the host machine (not in a container)
// This solves the Docker-in-Docker networking problem by running Maven + Testcontainers
// directly on the host where PostgreSQL containers are accessible via localhost
//
// CRITICAL: This uses exec.Command to run Maven on the actual host machine, bypassing Dagger containers entirely.
// Dagger's client.Host() API doesn't support command execution, only mounting resources INTO containers.
func (p *RailwayPipeline) runIntegrationTestsOnHost(ctx context.Context, client *dagger.Client, source *dagger.Directory) error {
	// Print stage header
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  STAGE: Integration Tests Execution (Host Machine)                           ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("📍 Location: Host machine (NOT in Dagger container)")
	fmt.Println("🐘 Testcontainers: Will use host Docker directly")
	fmt.Println("🔧 Tool: Maven Wrapper (../railway_framework/mvnw) - no Maven installation required")
	fmt.Println("")

	// Determine the working directory (where railway_framework and mvnw are located)
	// Get current working directory and construct absolute path
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// When running from dagger_go/, the railway_framework is in ../railway_framework
	// Use absolute path to avoid any ambiguity
	workDir := cwd + "/../railway_framework"

	fmt.Println("⚙️  Configuration:")
	fmt.Printf("   • Current Directory: %s\n", cwd)
	fmt.Printf("   • Working Directory: %s\n", workDir)
	fmt.Printf("   • Test Pattern: *IntegrationTest\n")
	fmt.Printf("   • Maven Profile: include-integration-tests\n")
	fmt.Printf("   • Java Version: %s (with preview features)\n", mavenReleaseVersion)
	fmt.Println("")

	// Run Maven wrapper test directly on host using exec.Command
	fmt.Println("🏃 Executing: ./mvnw test -Pinclude-integration-tests -Dtest=*IntegrationTest")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")

	cmd := exec.CommandContext(ctx, "./mvnw", "test",
		"-Pinclude-integration-tests", // Activate profile to include integration tests
		"-Dtest=*IntegrationTest",     // Run only integration test classes
		mavenCompilerRelease+mavenReleaseVersion,
		mavenCompilerArgs+mavenCompilerPreviewFlag)

	cmd.Dir = workDir // Set working directory

	// Capture output to parse test results while still showing it
	var outputBuffer strings.Builder
	multiWriter := io.MultiWriter(os.Stdout, &outputBuffer)
	cmd.Stdout = multiWriter
	cmd.Stderr = os.Stderr

	// Execute the command and wait for completion
	start := time.Now()
	err = cmd.Run() // err already declared above
	duration := time.Since(start)

	fmt.Println(separatorLine)

	// Parse and display test summary
	p.displayIntegrationTestSummary(outputBuffer.String(), duration, err)

	if err != nil {
		return fmt.Errorf("integration tests failed: %w", err)
	}

	return nil
}

// displayIntegrationTestSummary parses Maven test output and displays a summary
// similar to IntelliJ's test runner output
func (p *RailwayPipeline) displayIntegrationTestSummary(output string, duration time.Duration, testErr error) {
	// Parse test execution lines from Maven Surefire output
	// Patterns to match display names and test results

	// Pattern 1: [INFO] Running com.example.SomeIntegrationTest (fully qualified class name)
	runningClassPattern := regexp.MustCompile(`(?:\[INFO\]\s+)?Running (.+IntegrationTest)`)

	// Pattern 2: [INFO] Tests run: X, Failures: Y, Errors: Z, Skipped: W, Time elapsed: N s -- in DisplayName
	// This captures the display name used in output
	resultWithNamePattern := regexp.MustCompile(`Tests run: (\d+), Failures: (\d+), Errors: (\d+), Skipped: (\d+).* -- in (.+)$`)

	// Pattern 3: Standalone test results (fallback)
	resultPattern := regexp.MustCompile(`Tests run: (\d+), Failures: (\d+), Errors: (\d+), Skipped: (\d+)`)

	lines := strings.Split(output, "\n")
	var testResults []struct {
		name     string
		passed   bool
		failures int
		errors   int
	}

	// Parse test results
	var currentTest string
	for _, line := range lines {
		// Try to match running pattern (captures class name)
		if matches := runningClassPattern.FindStringSubmatch(line); matches != nil {
			currentTest = matches[1]
		}

		// Try to match result with display name (preferred - has test name)
		if matches := resultWithNamePattern.FindStringSubmatch(line); matches != nil {
			testName := matches[5] // Display name from "-- in <name>"
			failures := matches[2]
			errors := matches[3]
			passed := failures == "0" && errors == "0"
			failureCount := 0
			errorCount := 0
			fmt.Sscanf(failures, "%d", &failureCount)
			fmt.Sscanf(errors, "%d", &errorCount)

			testResults = append(testResults, struct {
				name     string
				passed   bool
				failures int
				errors   int
			}{
				name:     testName,
				passed:   passed,
				failures: failureCount,
				errors:   errorCount,
			})
			currentTest = "" // Reset
			continue
		}

		// Fallback: match result without name (use currentTest if available)
		if matches := resultPattern.FindStringSubmatch(line); matches != nil && currentTest != "" {
			failures := matches[2]
			errors := matches[3]
			passed := failures == "0" && errors == "0"
			failureCount := 0
			errorCount := 0
			fmt.Sscanf(failures, "%d", &failureCount)
			fmt.Sscanf(errors, "%d", &errorCount)

			testResults = append(testResults, struct {
				name     string
				passed   bool
				failures int
				errors   int
			}{
				name:     currentTest,
				passed:   passed,
				failures: failureCount,
				errors:   errorCount,
			})
			currentTest = ""
		}
	}

	// Display summary header
	fmt.Println("")
	fmt.Println("📊 Integration Test Summary")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")

	if len(testResults) == 0 {
		// Try to extract summary from Maven output
		summaryPattern := regexp.MustCompile(`Tests run: (\d+), Failures: (\d+), Errors: (\d+), Skipped: (\d+)`)
		var totalRun, totalFailures, totalErrors, totalSkipped int
		for _, line := range lines {
			if matches := summaryPattern.FindStringSubmatch(line); matches != nil {
				fmt.Sscanf(matches[1], "%d", &totalRun)
				fmt.Sscanf(matches[2], "%d", &totalFailures)
				fmt.Sscanf(matches[3], "%d", &totalErrors)
				fmt.Sscanf(matches[4], "%d", &totalSkipped)
			}
		}

		if totalRun > 0 {
			fmt.Printf("   Tests executed: %d (Failures: %d, Errors: %d, Skipped: %d)\n", totalRun, totalFailures, totalErrors, totalSkipped)
		} else {
			fmt.Println("   No individual test results parsed")
			fmt.Println("   Maven may be running in quiet mode or output format changed")
		}

		if testErr != nil {
			fmt.Printf("❌ FAILED: Integration tests failed after %v\n", duration)
			fmt.Printf("   Error: %v\n", testErr)
		} else {
			fmt.Printf("✅ SUCCESS: Integration tests passed in %v\n", duration)
		}
		return
	}

	// Display individual test results
	passedCount := 0
	failedCount := 0
	for _, result := range testResults {
		if result.passed {
			fmt.Printf("   ✅ %s\n", result.name)
			passedCount++
		} else {
			fmt.Printf("   ❌ %s (Failures: %d, Errors: %d)\n", result.name, result.failures, result.errors)
			failedCount++
		}
	}

	// Display overall summary
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	totalTests := passedCount + failedCount
	if failedCount == 0 {
		fmt.Printf("✅ SUCCESS: All %d integration tests passed in %v\n", totalTests, duration)
	} else {
		fmt.Printf("❌ FAILED: %d/%d integration tests failed after %v\n", failedCount, totalTests, duration)
		if testErr != nil {
			fmt.Printf("   Error: %v\n", testErr)
		}
	}
	fmt.Println("")
}

// parseEnvBool parses boolean environment variables with default value
// Accepts: true, True, TRUE, 1, yes, Yes, YES
// Everything else is treated as false
func parseEnvBool(envVar string, defaultValue bool) bool {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}

	lowerValue := strings.ToLower(value)
	return lowerValue == "true" || lowerValue == "1" || lowerValue == "yes"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
