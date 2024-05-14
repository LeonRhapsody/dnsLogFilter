# DNS Log Filter

DNS Log Filter is a simple tool designed to filter DNS logs for better analysis and monitoring. It processes DNS log files and extracts useful information, helping users to understand DNS traffic patterns, detect anomalies, and improve security posture.

## Features

- **Log Parsing**: Parses DNS log files in various formats, including standard DNS log formats.
- **Filtering**: Filters DNS logs based on user-defined criteria such as domain name, IP address, query type, etc.
- **Output Formats**: Supports multiple output formats including plain text, JSON, CSV, etc.
- **Customization**: Provides flexible configuration options to customize filtering rules and output formats.
- **Scalability**: Designed to handle large volumes of DNS log data efficiently.
- **Easy Integration**: Can be easily integrated into existing logging and monitoring pipelines.

## Installation

DNS Log Filter is written in [Go](https://golang.org/), so make sure you have Go installed on your system.

```sh
go get -u github.com/LeonRhapsody/dnsLogFilte
```

## Configuration
The DNS Log Filter can be configured using a configuration file. Below is an example configuration file:

```yaml
# Configuration example for DNS Log Filter

eth_name: "en0"                # Business network card name (used to get the local IP, define the file name)
analyze_threads: 1             # Number of analysis threads (adjust from small to large)
input_dir: "./input"           # Log input directory
input_format: "r,12,3,4,1,2,5,6,7,14,19,15,13" # Specify the format of the input logs
backup_dir: "./backup"         # Directory to move logs after processing
online_mode: true              # Online analysis/offline analysis

task_infos:
    apt:
        enable: true
        filter_ip_ruler:        # IP filter rules
        filter_domain_ruler:    # Domain filter rules
            - "./apt.list"
        output_dir: "./data/apt" # Output directory for results
        output_format: "6,12,17,18" # Output format
        is_gzip: true           # Whether to gzip the output file
        file_max_size: 200M     # Maximum file size (default is 200M)
        file_max_time: 1m       # Maximum file time (default is 1 minute)

    aliyun:
        enable: false
        filter_ip_ruler:
            - "ip_rule_1.txt"
        output_dir: "./data/aliyun"
        output_format: "jituan"
        is_upload: false
        is_gzip: false
        file_max_size: 200M
        file_max_time: 1m
```
## Usage

```sh
./dnslogfilter 
```

## Option

## Examples

```sh
./dnslogfilter 
```

This command will filter the DNS log file use the `config.yaml` file.

## Contributing


## License

