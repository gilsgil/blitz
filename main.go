package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Resolve the hostname with retries and delay. Returns empty string on failure.
func resolveHostname(hostname string, retries int, delay time.Duration) (string, error) {
	for i := 0; i <= retries; i++ {
		ips, err := net.LookupIP(hostname)
		if err == nil && len(ips) > 0 {
			return ips[0].String(), nil
		}
		if i < retries {
			time.Sleep(delay)
		}
	}
	return "", nil // Suppress error messages for hostname resolution failures
}

// Scan a single port on the host
func scanPort(host, displayHost string, port int, timeout time.Duration, wg *sync.WaitGroup, results chan<- string) {
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err == nil {
		conn.Close()
		results <- fmt.Sprintf("%s:%d", displayHost, port)
	}
}

// Scan all the ports for a specific host
func scanHost(host, displayHost string, ports []int, concurrency int, timeout time.Duration, wg *sync.WaitGroup, results chan<- string, sem chan struct{}) {
	defer wg.Done()
	hostWg := &sync.WaitGroup{}
	semHost := make(chan struct{}, concurrency)

	for _, port := range ports {
		hostWg.Add(1)
		semHost <- struct{}{}
		go func(port int) {
			defer func() { <-semHost }()
			scanPort(host, displayHost, port, timeout, hostWg, results)
		}(port)
	}

	hostWg.Wait()
	<-sem // Release the thread for the host
}

// Parse the port specification and return a list of ports
func parsePorts(portSpec string) ([]int, error) {
	var ports []int
	ranges := strings.Split(portSpec, ",")
	for _, r := range ranges {
		if strings.Contains(r, "-") {
			bounds := strings.Split(r, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid port range: %s", r)
			}
			start, err := strconv.Atoi(bounds[0])
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", r)
			}
			end, err := strconv.Atoi(bounds[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", r)
			}
			for i := start; i <= end; i++ {
				ports = append(ports, i)
			}
		} else {
			port, err := strconv.Atoi(r)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", r)
			}
			ports = append(ports, port)
		}
	}
	return ports, nil
}

// Expand CIDR block into list of IPs
func expandCIDR(cidr string) ([]string, error) {
	var ips []string
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		ips = append(ips, ip.String())
	}
	return ips, nil
}

// Increment an IP address
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func main() {
	target := flag.String("t", "", "Target IP or hostname")
	cidr := flag.String("cidr", "", "CIDR block of IPs (e.g., 192.168.1.0/24)")
	listCIDR := flag.String("lcidr", "", "List of CIDR blocks (file or stdin)")
	list := flag.String("l", "", "List of hosts/IPs (file)")
	concurrency := flag.Int("c", 10, "Number of parallel threads")
	portSpec := flag.String("p", "80,443", "Ports to scan (e.g., 80,443 or 1-1000)")
	timeout := flag.Int("timeout", 2, "Timeout in seconds for checking each port")
	retries := flag.Int("retries", 3, "Number of retries to resolve the hostname")
	delay := flag.Int("delay", 1, "Delay in seconds between hostname resolution retries")
	clean := flag.Bool("clean", false, "Run cleanports.py after scan to keep only ports 80 and 443 for domains with more than 20 open ports")
	output := flag.String("o", "", "Specify output file path for cleaned results (required if -clean is passed)")
	flag.Parse()

	var hosts []string
	if *target != "" {
		hosts = append(hosts, *target)
	} else if *cidr != "" {
		expandedHosts, err := expandCIDR(*cidr)
		if err != nil {
			fmt.Println("Error expanding CIDR:", err)
			return
		}
		hosts = append(hosts, expandedHosts...)
	} else if *listCIDR != "" {
		// Handle listCIDR...
	} else if *list != "" {
		// Handle list...
	}

	ports, err := parsePorts(*portSpec)
	if err != nil {
		fmt.Println("Error parsing ports:", err)
		return
	}

	resultsFile := "scan_results.txt" // File to store the scan results
	results := make(chan string)
	timeoutDuration := time.Duration(*timeout) * time.Second
	sem := make(chan struct{}, *concurrency) // Control the overall concurrency

	go func() {
		f, _ := os.Create(resultsFile)
		defer f.Close()

		for result := range results {
			fmt.Println(result)
			f.WriteString(result + "\n") // Save results to file
		}
	}()

	wg := &sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)
		displayHost := host
		ip := host
		if net.ParseIP(host) == nil {
			resolvedIP, err := resolveHostname(host, *retries, time.Duration(*delay)*time.Second)
			if err != nil {
				wg.Done()
				continue
			}
			ip = resolvedIP
		}
		sem <- struct{}{} // Control the number of active hosts
		go scanHost(ip, displayHost, ports, *concurrency, timeoutDuration, wg, results, sem)
	}

	wg.Wait()
	close(results)

	// If -clean flag is passed, run clean_ports.py
	if *clean {
		if *output == "" {
			fmt.Println("Error: -o output file must be specified when using the -clean option.")
			return
		}
		cleanPortsAfterScan(resultsFile, *output, 20) // Example: Keeping only 80 and 443 for domains with more than 20 ports
	}
}

// Function to call the Python script for cleaning ports
func cleanPortsAfterScan(inputFile string, outputFile string, portLimit int) {
	// Call the Python script to clean the scan results
	cmd := exec.Command("python3", "cleanports.py", "-f", inputFile, "-l", strconv.Itoa(portLimit), "-o", outputFile)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error running cleanports.py:", err)
	}
}
