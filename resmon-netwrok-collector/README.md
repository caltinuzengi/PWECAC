# Resmon Collector

Resmon Collector is a lightweight and efficient Prometheus exporter that collects network resource metrics from the Windows Resource Monitor (`resmon`) and exposes them in a Prometheus-compatible format. This allows for seamless integration into your observability stack, providing valuable insights into network usage and performance on Windows systems.

## Features

- **Real-Time Metrics**: Collects real-time network resource metrics such as bandwidth usage, packet counts, and connection statistics.
- **Prometheus-Compatible**: Exposes metrics in a format compatible with Prometheus for easy scraping.
- **Lightweight**: Designed to have minimal impact on system performance.
- **Easy to Use**: Simple configuration and deployment process.

## Prerequisites

- **Windows OS**: The collector is designed to work on Windows systems.
- **Prometheus**: Ensure Prometheus is set up and running in your environment.
- **Administrator Privileges**: Required to access `resmon` metrics.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/resmon-collector.git
   ```
2. Navigate to the project directory:
   ```bash
   cd resmon-collector
   ```
3. Build the project (if applicable):
   ```bash
   go build -o resmon-collector main.go
   ```
4. Run the exporter:
   ```bash
   ./resmon-collector
   ```

## Usage

By default, Resmon Collector runs on port `9182`. You can access the metrics endpoint at:

```
http://<your-windows-host>:9182/metrics
```

Example output:
```
# HELP resmon_network_bytes_received Total bytes received
# TYPE resmon_network_bytes_received counter
resmon_network_bytes_received{interface="Ethernet"} 1234567

# HELP resmon_network_bytes_sent Total bytes sent
# TYPE resmon_network_bytes_sent counter
resmon_network_bytes_sent{interface="Ethernet"} 7654321
```

<!-- ## Configuration

You can configure the collector by editing the `config.yaml` file. Example:

```yaml
port: 9183
log_level: info
interfaces:
  - Ethernet
  - Wi-Fi
``` -->

## Prometheus Configuration

Add the following scrape job to your Prometheus configuration file:

```yaml
scrape_configs:
  - job_name: 'resmon-collector'
    static_configs:
      - targets: ['<your-windows-host>:9183']
```

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

For questions or support, please contact [your-email@example.com](mailto:your-email@example.com).