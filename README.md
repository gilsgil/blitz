
# blitz - Ultra-Fast Port Scanner with CDN Detection

`blitz` is an ultra-fast port scanning tool designed to handle massive amounts of hosts and IP ranges efficiently. The tool can detect CDN providers such as Akamai, Imperva, and Cloudflare, allowing users to filter out domains protected by these CDNs. It can also perform highly concurrent scans, making it suitable for rapid port enumeration on large IP blocks or host lists.

## Features

- **Ultra-Fast Scanning:** Supports high concurrency for rapid port scans. For optimal performance, use `-c 250` for maximum concurrency.
- **CDN Detection:** Automatically detects and excludes domains behind CDNs like Akamai, Imperva, Cloudflare, Fastly, and others using DNS and HTTP methods.
- **Port Range Scanning:** Scan specific ports or port ranges using a customizable port specification.
- **CIDR Expansion:** Automatically expands CIDR blocks to IP ranges for efficient network scanning.
- **Hostname Resolution with Retries:** Resolves hostnames with multiple retry attempts for reliable scanning.
- **Randomized Concurrency:** Controls how many scans run in parallel to optimize network utilization.
- **Silent Mode:** Suppresses error messages for failed hostname resolutions, allowing for cleaner outputs.

## Installation

### Prerequisites

- **Go 1.16+** installed on your system.

### Install with Go

```bash
go install github.com/yourusername/blitz@latest
```

This will install the latest version of `blitz` and make it available globally via the command line.

## Usage

blitz can be used to scan a single target, a list of hosts, or an entire IP range defined by a CIDR block. 

### Command-Line Options

- `-t`, `--target`: Specify a single target IP or hostname.
- `-cidr`: Specify a CIDR block of IPs (e.g., `192.168.1.0/24`).
- `-lcidr`: Specify a list of CIDR blocks from a file or stdin.
- `-l`, `--list`: Specify a list of hosts or IPs from a file.
- `-c`, `--concurrence`: Number of concurrent scans (default: 10). **Recommended value for high performance: 250**.
- `-p`, `--ports`: Specify ports to scan (e.g., `80,443` or `1-1000`).
- `--timeout`: Timeout in seconds for each port check (default: 2 seconds).
- `--retries`: Number of retries for hostname resolution (default: 3 retries).
- `--delay`: Delay between hostname resolution retries in seconds (default: 1 second).

### Example Usage

1. **Scan a single target with default ports (80, 443):**

   ```bash
   blitz -t example.com
   ```

2. **Scan a CIDR block for specific ports:**

   ```bash
   blitz --cidr 192.168.1.0/24 -p 22,80,443
   ```

3. **Scan a list of hosts with high concurrency (250):**

   ```bash
   blitz -l hosts.txt -c 250
   ```

4. **Scan a CIDR block list from a file:**

   ```bash
   blitz --lcidr cidr_list.txt -p 80,443
   ```

5. **Use stdin for CIDR or host input:**

   ```bash
   cat cidr_list.txt | blitz --lcidr -
   ```

## Performance Tips

- **Increase Concurrency:** For ultra-fast scanning, use the `-c` flag to set the number of concurrent threads. The recommended value is `-c 250` for optimal speed.
- **Limit Port Range:** Scanning fewer ports (e.g., `80,443`) can significantly speed up the process.
- **CIDR Expansion:** Use the `--cidr` option for automatic IP range expansion when scanning networks.

## Output

The tool outputs active hosts and open ports in the format: `hostname:port`. Domains that are behind CDNs are filtered out by default unless verbose mode is enabled.

## Contact

For any questions or feedback, feel free to reach out via email or open an issue on GitHub.
