package com.border.simulator;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * PRT Services Simulator — entry point.
 *
 * <p>Starts an embedded web server exposing 3 mock REST endpoints
 * that simulate the real authentication and certificate download services
 * used by cert-parser.
 */
@SpringBootApplication
public class PrtServicesSimulatorApplication {

    public static void main(String[] args) {
        SpringApplication.run(PrtServicesSimulatorApplication.class, args);
    }
}
