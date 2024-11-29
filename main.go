package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

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
	return "", fmt.Errorf("hostname não resolvido: %s", hostname)
}

func scanPort(host, displayHost string, port int, timeout time.Duration, wg *sync.WaitGroup, results chan<- string) {
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err == nil {
		conn.Close()
		results <- fmt.Sprintf("%s:%d", displayHost, port)
	}
}

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

func parsePorts(portSpec string) ([]int, error) {
	var ports []int
	ranges := strings.Split(portSpec, ",")
	for _, r := range ranges {
		if strings.Contains(r, "-") {
			bounds := strings.Split(r, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("formato inválido de intervalo de portas: %s", r)
			}
			start, err := strconv.Atoi(bounds[0])
			if err != nil {
				return nil, fmt.Errorf("portas inválidas: %s", r)
			}
			end, err := strconv.Atoi(bounds[1])
			if err != nil {
				return nil, fmt.Errorf("portas inválidas: %s", r)
			}
			for i := start; i <= end; i++ {
				ports = append(ports, i)
			}
		} else {
			port, err := strconv.Atoi(r)
			if err != nil {
				return nil, fmt.Errorf("porta inválida: %s", r)
			}
			ports = append(ports, port)
		}
	}
	return ports, nil
}

// Função para expandir o bloco CIDR em uma lista de IPs
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

// Função para incrementar um IP
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func main() {
	target := flag.String("t", "", "Target IP ou hostname")
	cidr := flag.String("cidr", "", "Bloco de IPs em notação CIDR (ex: 192.168.1.0/24)")
	listCIDR := flag.String("lcidr", "", "Lista de blocos CIDR (arquivo ou stdin)")
	list := flag.String("l", "", "Lista de hosts/IPs (arquivo)")
	concurrency := flag.Int("c", 10, "Número de threads paralelas")
	portSpec := flag.String("p", "80,443", "Portas a escanear (ex: 80,443 ou 1-1000)")
	timeout := flag.Int("timeout", 2, "Timeout em segundos para a checagem de cada porta")
	retries := flag.Int("retries", 3, "Número de tentativas para resolver o hostname")
	delay := flag.Int("delay", 1, "Tempo em segundos entre as tentativas de resolução de hostname")
	flag.Parse()

	var hosts []string
	if *target != "" {
		hosts = append(hosts, *target)
	} else if *cidr != "" {
		expandedHosts, err := expandCIDR(*cidr)
		if err == nil {
			hosts = append(hosts, expandedHosts...)
		}
	} else if *listCIDR != "" {
		file, err := os.Open(*listCIDR)
		var scanner *bufio.Scanner
		if err != nil {
			// Se não conseguir abrir o arquivo, considera stdin
			scanner = bufio.NewScanner(os.Stdin)
		} else {
			defer file.Close()
			scanner = bufio.NewScanner(file)
		}

		for scanner.Scan() {
			line := scanner.Text()
			expandedHosts, err := expandCIDR(line)
			if err == nil {
				hosts = append(hosts, expandedHosts...)
			}
		}
	} else if *list != "" {
		file, err := os.Open(*list)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				hosts = append(hosts, scanner.Text())
			}
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			hosts = append(hosts, scanner.Text())
		}
	}

	ports, err := parsePorts(*portSpec)
	if err != nil {
		return
	}

	results := make(chan string)
	timeoutDuration := time.Duration(*timeout) * time.Second
	sem := make(chan struct{}, *concurrency) // Control the overall concurrency

	go func() {
		for result := range results {
			fmt.Println(result)
		}
	}()

	wg := &sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)
		displayHost := host
		ip := host
		if net.ParseIP(host) == nil {
			resolvedIP, err := resolveHostname(host, *retries, time.Duration(*delay)*time.Second)
			if err == nil {
				if resolvedIP == "127.0.0.1" || resolvedIP == "127.0.0.2" || resolvedIP == "127.0.0.3" { // Ignorar hosts que resolvem para 127.0.0.1
					wg.Done()
					continue
				}
				ip = resolvedIP
			} else {
				wg.Done()
				continue
			}
		}
		sem <- struct{}{} // Control the number of active hosts
		go scanHost(ip, displayHost, ports, *concurrency, timeoutDuration, wg, results, sem)
	}

	wg.Wait()
	close(results)
}
