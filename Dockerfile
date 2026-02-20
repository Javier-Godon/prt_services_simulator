# ═══════════════════════════════════════════════════════════════
# PRT Services Simulator — Multi-stage Docker build
# ═══════════════════════════════════════════════════════════════
# Stage 1: Build with Maven + Amazon Corretto JDK 25
# Stage 2: Minimal runtime with Corretto JRE 25
# Result: ~300MB image (vs ~1GB single-stage)
# ═══════════════════════════════════════════════════════════════

# ── Stage 1: Build ────────────────────────────────────────────
FROM amazoncorretto:25.0.1 AS builder

WORKDIR /build

# Install Maven
RUN yum install -y maven && yum clean all

# Copy POM first for dependency caching
COPY pom.xml .
RUN mvn dependency:go-offline -q \
    -Dmaven.compiler.release=25 \
    -Dmaven.compiler.compilerArgs=--enable-preview

# Copy source and build
COPY src/ src/
RUN mvn package -DskipTests -q \
    -Dmaven.compiler.release=25 \
    -Dmaven.compiler.compilerArgs=--enable-preview

# ── Stage 2: Runtime ─────────────────────────────────────────
FROM amazoncorretto:25.0.1-alpine AS runtime

# Labels for GitHub Container Registry
LABEL org.opencontainers.image.source="https://github.com/Javier-Godon/prt_services_simulator"
LABEL org.opencontainers.image.description="PRT Services Simulator — mock REST endpoints for cert-parser"
LABEL org.opencontainers.image.licenses="MIT"

# Non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the fat JAR from builder
COPY --from=builder /build/target/*.jar app.jar

# Spring Boot actuator port
EXPOSE 8087

# Run as non-root
USER appuser

# JVM flags: enable preview features, optimize for containers
ENTRYPOINT ["java", \
    "--enable-preview", \
    "-XX:+UseContainerSupport", \
    "-XX:MaxRAMPercentage=75.0", \
    "-jar", "app.jar"]
