package com.border.simulator.controller;

import com.border.simulator.config.SimulatorProperties;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

/**
 * Step 1: Simulated OpenID Connect token endpoint.
 *
 * <p>Accepts POST requests with OAuth2 password grant parameters
 * and returns a JSON response containing an access_token.
 *
 * <p>Validates: grant_type, client_id, client_secret, username, password.
 * Returns 401 if any credential is invalid.
 */
@RestController
public class AuthTokenController {

    private static final Logger log = LoggerFactory.getLogger(AuthTokenController.class);

    private final SimulatorProperties.AuthProperties authProps;

    public AuthTokenController(SimulatorProperties properties) {
        this.authProps = properties.auth();
    }

    @PostMapping(
            path = "/protocol/openid-connect/token",
            consumes = MediaType.APPLICATION_FORM_URLENCODED_VALUE,
            produces = MediaType.APPLICATION_JSON_VALUE
    )
    public ResponseEntity<Map<String, String>> token(
            @RequestParam("grant_type") String grantType,
            @RequestParam("client_id") String clientId,
            @RequestParam("client_secret") String clientSecret,
            @RequestParam("username") String username,
            @RequestParam("password") String password
    ) {
        log.info("Token request: grant_type={}, client_id={}, username={}", grantType, clientId, username);

        if (!"password".equals(grantType)) {
            log.warn("Rejected: invalid grant_type={}", grantType);
            return ResponseEntity.badRequest().body(Map.of("error", "unsupported_grant_type"));
        }

        if (!authProps.expectedClientId().equals(clientId)
                || !authProps.expectedClientSecret().equals(clientSecret)
                || !authProps.expectedUsername().equals(username)
                || !authProps.expectedPassword().equals(password)) {
            log.warn("Rejected: invalid credentials for client_id={}, username={}", clientId, username);
            return ResponseEntity.status(401).body(Map.of("error", "invalid_grant"));
        }

        log.info("Token issued: access_token={}", authProps.accessToken());
        return ResponseEntity.ok(Map.of(
                "access_token", authProps.accessToken(),
                "token_type", "Bearer",
                "expires_in", "3600"
        ));
    }
}
