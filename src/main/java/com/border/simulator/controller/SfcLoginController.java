package com.border.simulator.controller;

import com.border.simulator.config.SimulatorProperties;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;
import java.util.Objects;

/**
 * Step 2: Simulated SFC login endpoint.
 *
 * <p>Accepts POST requests with a Bearer access token in the Authorization header
 * and a JSON body containing border post configuration.
 * Returns the SFC token as plain text.
 *
 * <p>Validates: access_token in header, borderPostId/boxId/passengerControlType in body.
 * Returns 401 if the access token is invalid.
 */
@RestController
public class SfcLoginController {

    private static final Logger log = LoggerFactory.getLogger(SfcLoginController.class);

    private final SimulatorProperties.AuthProperties authProps;
    private final SimulatorProperties.LoginProperties loginProps;

    public SfcLoginController(SimulatorProperties properties) {
        this.authProps = properties.auth();
        this.loginProps = properties.login();
    }

    @PostMapping(
            path = "/auth/v1/login",
            consumes = MediaType.APPLICATION_JSON_VALUE,
            produces = MediaType.TEXT_PLAIN_VALUE
    )
    public ResponseEntity<String> login(
            @RequestHeader("Authorization") String authHeader,
            @RequestBody Map<String, Object> body
    ) {
        log.info("SFC login request: body={}", body);

        // Validate Bearer token
        String expectedBearer = "Bearer " + authProps.accessToken();
        if (!expectedBearer.equals(authHeader)) {
            log.warn("Rejected: invalid access token");
            return ResponseEntity.status(401).body("Invalid access token");
        }

        // Validate border post config (optional — log mismatches but still succeed)
        int borderPostId = ((Number) body.getOrDefault("borderPostId", 0)).intValue();
        String boxId = String.valueOf(body.getOrDefault("boxId", ""));
        int passengerControlType = ((Number) body.getOrDefault("passengerControlType", 0)).intValue();

        if (borderPostId != loginProps.expectedBorderPostId()
                || !Objects.equals(boxId, loginProps.expectedBoxId())
                || passengerControlType != loginProps.expectedPassengerControlType()) {
            log.warn("Border post config mismatch: got ({}, {}, {}), expected ({}, {}, {})",
                    borderPostId, boxId, passengerControlType,
                    loginProps.expectedBorderPostId(), loginProps.expectedBoxId(),
                    loginProps.expectedPassengerControlType());
        }

        log.info("SFC token issued: {}", loginProps.sfcToken());
        return ResponseEntity.ok(loginProps.sfcToken());
    }
}
