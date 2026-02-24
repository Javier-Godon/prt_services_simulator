package com.border.simulator.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * Type-safe configuration for the simulator, bound to {@code simulator.*} in application.yaml.
 *
 * <p>Defines the expected credentials and tokens for each simulated endpoint.
 * Override values via environment variables for Kubernetes deployment.
 */
@ConfigurationProperties(prefix = "simulator")
public record SimulatorProperties(
        AuthProperties auth,
        LoginProperties login,
        DownloadProperties download
) {

    /**
     * Step 1: OpenID Connect password grant configuration.
     */
    public record AuthProperties(
            String expectedClientId,
            String expectedClientSecret,
            String expectedUsername,
            String expectedPassword,
            String accessToken
    ) {}

    /**
     * Step 2: SFC login configuration.
     */
    public record LoginProperties(
            String expectedBorderPostId,
            String expectedBoxId,
            String expectedPassengerControlType,
            String sfcToken
    ) {}

    /**
     * Step 3: Certificate download configuration.
     */
    public record DownloadProperties(
            String fixtureFile
    ) {}
}
