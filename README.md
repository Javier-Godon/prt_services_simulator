# PRT Services Simulator

Mock REST services simulating the 3-endpoint authentication and certificate download flow used by **cert-parser**.

## Endpoints

| Step | Method | Path | Description |
|------|--------|------|-------------|
| 1 | POST | `/protocol/openid-connect/token` | OpenID Connect password grant → access_token |
| 2 | POST | `/auth/v1/login` | SFC login with bearer + JSON body → sfc_token |
| 3 | GET | `/certificates/csca` | Certificate download with dual-token auth → .bin file |

## Authentication Flow

```
cert-parser                              prt_services_simulator
─────────────                            ──────────────────────

POST /protocol/openid-connect/token
  grant_type=password
  client_id=...                    ──►   Validates credentials
  client_secret=...                      Returns {"access_token": "..."}
  username=...                     ◄──
  password=...

POST /auth/v1/login
  Authorization: Bearer {access_token}
  {"borderPostId": 1,              ──►   Validates bearer token
   "boxId": 1,                           Returns SFC token as text
   "passengerControlType": 1}      ◄──

GET /certificates/csca
  Authorization: Bearer {access_token}
  x-sfc-authorization: Bearer      ──►   Validates both tokens
    {sfc_token}                           Returns .bin fixture file
                                   ◄──
```

## Configuration

All settings are in `src/main/resources/application.yaml`. Override via environment variables:

```bash
# Override expected credentials
SIMULATOR_AUTH_EXPECTED_CLIENT_ID=my-client
SIMULATOR_AUTH_EXPECTED_USERNAME=my-user
SIMULATOR_AUTH_ACCESS_TOKEN=custom-token

# Override border post config
SIMULATOR_LOGIN_EXPECTED_BORDER_POST_ID=42

# Override fixture file
SIMULATOR_DOWNLOAD_FIXTURE_FILE=fixtures/custom.bin
```

## Build & Run

```bash
# Build
./mvnw clean package

# Run
./mvnw spring-boot:run

# Run tests
./mvnw test
```

## Tech Stack

- Java 25
- Spring Boot 3.5 (latest stable, Spring Boot 4 not yet released)
- Maven
- No external dependencies beyond Spring Boot Starter Web

## Project Structure

```
src/main/java/com/border/simulator/
├── PrtServicesSimulatorApplication.java    # Entry point
├── config/
│   ├── SimulatorConfig.java               # Enables config properties
│   └── SimulatorProperties.java           # Type-safe YAML binding
└── controller/
    ├── AuthTokenController.java           # Step 1: OpenID Connect
    ├── SfcLoginController.java            # Step 2: SFC login
    └── CertificateDownloadController.java # Step 3: Certificate download
```
