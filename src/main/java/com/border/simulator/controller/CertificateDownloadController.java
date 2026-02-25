package com.border.simulator.controller;

import com.border.simulator.config.SimulatorProperties;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.core.io.ClassPathResource;
import org.springframework.core.io.FileSystemResource;
import org.springframework.core.io.Resource;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

import java.io.IOException;

/**
 * Step 3: Simulated certificate download endpoint.
 *
 * <p>Accepts GET requests with dual-token authentication:
 * <ul>
 *   <li>{@code Authorization: Bearer {access_token}}</li>
 *   <li>{@code x-sfc-authorization: Bearer {sfc_token}}</li>
 * </ul>
 *
 * <p>Returns a .bin fixture file (CMS/PKCS#7 Master List) as binary content.
 * Returns 401 if either token is invalid.
 */
@RestController
public class CertificateDownloadController {

    private static final Logger log = LoggerFactory.getLogger(CertificateDownloadController.class);

    private final SimulatorProperties properties;

    public CertificateDownloadController(SimulatorProperties properties) {
        this.properties = properties;
    }

    @GetMapping(
            path = "/certificates/csca",
            produces = MediaType.APPLICATION_OCTET_STREAM_VALUE
    )
    public ResponseEntity<byte[]> download(
            @RequestHeader("Authorization") String authHeader,
            @RequestHeader("x-sfc-authorization") String sfcAuthHeader
    ) {
        log.info("Download request received");

        // Validate access token
        String expectedAuth = "Bearer " + properties.auth().accessToken();
        if (!expectedAuth.equals(authHeader)) {
            log.warn("Rejected: invalid access token in Authorization header");
            return ResponseEntity.status(401).body(null);
        }

        // Validate SFC token
        String expectedSfc = "Bearer " + properties.login().sfcToken();
        if (!expectedSfc.equals(sfcAuthHeader)) {
            log.warn("Rejected: invalid SFC token in x-sfc-authorization header");
            return ResponseEntity.status(401).body(null);
        }

        // Serve fixture file
        String fixturePath = properties.download().fixtureFile();
        try {
            // Support both classpath resources (relative paths like "fixtures/file.bin")
            // and filesystem paths (absolute paths like "/data/fixtures/file.bin" from K8s mounts)
            Resource resource;
            if (fixturePath.startsWith("/")) {
                resource = new FileSystemResource(fixturePath);
            } else {
                resource = new ClassPathResource(fixturePath);
            }
            byte[] data = resource.getInputStream().readAllBytes();
            log.info("Serving fixture: {} ({} bytes)", fixturePath, data.length);
            return ResponseEntity.ok()
                    .contentType(MediaType.APPLICATION_OCTET_STREAM)
                    .header("Content-Disposition", "attachment; filename=masterlist.bin")
                    .body(data);
        } catch (IOException e) {
            log.error("Fixture file not found: {}", fixturePath, e);
            return ResponseEntity.internalServerError().body(null);
        }
    }
}
