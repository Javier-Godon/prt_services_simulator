package com.border.simulator.config;

import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Configuration;

/**
 * Enables binding of {@link SimulatorProperties} from application.yaml.
 */
@Configuration
@EnableConfigurationProperties(SimulatorProperties.class)
public class SimulatorConfig {
}
