package com.border.simulator;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

/**
 * Integration tests for all 3 simulated endpoints.
 *
 * <p>Verifies the full authentication and download flow using MockMvc.
 * Tests both success and failure paths for each endpoint.
 */
@SpringBootTest
@AutoConfigureMockMvc
class PrtServicesSimulatorApplicationTests {

    @Autowired
    private MockMvc mockMvc;

    // ── Step 1: OpenID Connect Token ──────────────────────────────

    @Test
    void step1_validCredentials_returnsAccessToken() throws Exception {
        mockMvc.perform(post("/protocol/openid-connect/token")
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param("grant_type", "password")
                        .param("client_id", "cert-parser-client")
                        .param("client_secret", "super-secret-123")
                        .param("username", "operator@border.gov")
                        .param("password", "operator-pass"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.access_token").value("simulated-access-token-abc123"))
                .andExpect(jsonPath("$.token_type").value("Bearer"));
    }

    @Test
    void step1_invalidPassword_returns401() throws Exception {
        mockMvc.perform(post("/protocol/openid-connect/token")
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param("grant_type", "password")
                        .param("client_id", "cert-parser-client")
                        .param("client_secret", "super-secret-123")
                        .param("username", "operator@border.gov")
                        .param("password", "wrong-password"))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.error").value("invalid_grant"));
    }

    @Test
    void step1_invalidGrantType_returns400() throws Exception {
        mockMvc.perform(post("/protocol/openid-connect/token")
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param("grant_type", "client_credentials")
                        .param("client_id", "cert-parser-client")
                        .param("client_secret", "super-secret-123")
                        .param("username", "operator@border.gov")
                        .param("password", "operator-pass"))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("unsupported_grant_type"));
    }

    // ── Step 2: SFC Login ──────────────────────────────────────

    @Test
    void step2_validBearerAndBody_returnsSfcToken() throws Exception {
        mockMvc.perform(post("/auth/v1/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .header("Authorization", "Bearer simulated-access-token-abc123")
                        .content("""
                                {"borderPostId": 1, "boxId": 1, "passengerControlType": 1}
                                """))
                .andExpect(status().isOk())
                .andExpect(content().string("simulated-sfc-token-xyz789"));
    }

    @Test
    void step2_invalidBearer_returns401() throws Exception {
        mockMvc.perform(post("/auth/v1/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .header("Authorization", "Bearer wrong-token")
                        .content("""
                                {"borderPostId": 1, "boxId": 1, "passengerControlType": 1}
                                """))
                .andExpect(status().isUnauthorized());
    }

    // ── Step 3: Certificate Download ──────────────────────────

    @Test
    void step3_validDualTokens_returnsBinary() throws Exception {
        mockMvc.perform(get("/certificates/csca")
                        .header("Authorization", "Bearer simulated-access-token-abc123")
                        .header("x-sfc-authorization", "Bearer simulated-sfc-token-xyz789"))
                .andExpect(status().isOk())
                .andExpect(header().string("Content-Disposition", "attachment; filename=masterlist.bin"))
                .andExpect(content().contentType(MediaType.APPLICATION_OCTET_STREAM));
    }

    @Test
    void step3_invalidAccessToken_returns401() throws Exception {
        mockMvc.perform(get("/certificates/csca")
                        .header("Authorization", "Bearer wrong-access-token")
                        .header("x-sfc-authorization", "Bearer simulated-sfc-token-xyz789"))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void step3_invalidSfcToken_returns401() throws Exception {
        mockMvc.perform(get("/certificates/csca")
                        .header("Authorization", "Bearer simulated-access-token-abc123")
                        .header("x-sfc-authorization", "Bearer wrong-sfc-token"))
                .andExpect(status().isUnauthorized());
    }

    // ── Full Flow (Step 1 → Step 2 → Step 3) ──────────────────

    @Test
    void fullFlow_allThreeStepsSucceed() throws Exception {
        // Step 1: Get access token
        String tokenResponse = mockMvc.perform(post("/protocol/openid-connect/token")
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param("grant_type", "password")
                        .param("client_id", "cert-parser-client")
                        .param("client_secret", "super-secret-123")
                        .param("username", "operator@border.gov")
                        .param("password", "operator-pass"))
                .andExpect(status().isOk())
                .andReturn().getResponse().getContentAsString();

        // Extract access token (simple JSON parsing)
        String accessToken = tokenResponse.split("\"access_token\":\"")[1].split("\"")[0];

        // Step 2: Get SFC token
        String sfcToken = mockMvc.perform(post("/auth/v1/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .header("Authorization", "Bearer " + accessToken)
                        .content("""
                                {"borderPostId": 1, "boxId": 1, "passengerControlType": 1}
                                """))
                .andExpect(status().isOk())
                .andReturn().getResponse().getContentAsString();

        // Step 3: Download with both tokens
        mockMvc.perform(get("/certificates/csca")
                        .header("Authorization", "Bearer " + accessToken)
                        .header("x-sfc-authorization", "Bearer " + sfcToken))
                .andExpect(status().isOk())
                .andExpect(content().contentType(MediaType.APPLICATION_OCTET_STREAM));
    }
}
