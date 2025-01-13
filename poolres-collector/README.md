# PoolRes Collector

PoolRes Collector is a Prometheus exporter designed to monitor IIS (Internet Information Services) application pool processes and expose resource usage metrics such as CPU and memory consumption. This exporter provides granular visibility into individual IIS processes, enabling better resource management and performance monitoring.

## Features

- **Process-Level Metrics**: Collects CPU and memory usage for each process running under IIS application pools.
- **Prometheus-Compatible**: Exposes metrics in a format compatible with Prometheus for easy integration.
- **IIS-Focused Monitoring**: Specifically tailored for monitoring IIS application pool processes.
- **Lightweight and Efficient**: Minimal resource overhead on the monitored system.

## Prerequisites

- **Windows OS**: This collector is designed to work on systems running IIS.
- **Prometheus**: Ensure Prometheus is installed and running in your environment.
- **Administrator Privileges**: Required to access process-level metrics for IIS.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/poolres-collector.git
   ```
2. Navigate to the project directory:
   ```bash
   cd poolres-collector
   ```
3. Build the project (if applicable):
   ```bash
   go build -o poolres-collector main.go
   ```
4. Run the exporter:
   ```bash
   ./poolres-collector
   ```

## Usage

By default, PoolRes Collector runs on port `9183`. You can access the metrics endpoint at:

```
http://<your-windows-host>:9183/metrics
```

Example output:
```
# HELP poolres_cpu_usage_percent CPU usage percentage per process
# TYPE poolres_cpu_usage_percent gauge
poolres_cpu_usage_percent{pool="DefaultAppPool",pid="1234"} 12.5

# HELP poolres_memory_usage_bytes Memory usage in bytes per process
# TYPE poolres_memory_usage_bytes gauge
poolres_memory_usage_bytes{pool="DefaultAppPool",pid="1234"} 104857600
```

<!-- ## Configuration

You can configure the collector by editing the `config.yaml` file. Example:

```yaml
port: 9183
log_level: info
poll_interval: 10s
pools:
  - DefaultAppPool
  - MyCustomAppPool
``` -->

## Prometheus Configuration

Add the following scrape job to your Prometheus configuration file:

```yaml
scrape_configs:
  - job_name: 'poolres-collector'
    static_configs:
      - targets: ['<your-windows-host>:9183']
```

## Contributing

Contributions are welcome! Please feel free to open issues or submit pull requests to improve PoolRes Collector.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

For questions or support, please contact [cagrialtinuzengi@gmail.com](mailto:cagrialtinuzengi@gmail.com).
