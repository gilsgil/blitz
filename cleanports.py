import argparse
import os

def clean_ports(file_path, port_limit=20):
    cleaned_file = f"{file_path}_cleaned.txt"
    
    # Dictionary to count ports by domain
    domain_ports = {}
    
    # Read the file with open ports
    with open(file_path, 'r') as f:
        for line in f:
            # Separate the domain from the port using the first ":" as a separator
            parts = line.strip().split(':')
            if len(parts) < 2:
                continue  # Skip invalid lines
            domain = parts[0]
            port = parts[1]
            
            if domain not in domain_ports:
                domain_ports[domain] = []
            domain_ports[domain].append(port)
    
    # Write the cleaned file
    with open(cleaned_file, 'w') as f:
        for domain, ports in domain_ports.items():
            if len(ports) > port_limit:
                # Keep only ports 80 and 443
                if '80' in ports:
                    f.write(f"{domain}:80\n")
                if '443' in ports:
                    f.write(f"{domain}:443\n")
            else:
                # Keep all ports for domains with port_limit or fewer ports
                for port in ports:
                    f.write(f"{domain}:{port}\n")
    
    # Replace the original file with the cleaned one
    os.replace(cleaned_file, file_path)
    print(f"[*] Port cleaning completed. Only ports 80 and 443 were kept for domains with more than {port_limit} open ports.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Cleans ports from a file, keeping only ports 80 and 443 for domains with more than a specified limit of open ports.')
    parser.add_argument('-f', '--file', type=str, required=True, help='Path to the file with open ports in the format target.com:port')
    parser.add_argument('-l', '--limit', type=int, default=20, help='Port limit per domain before keeping only 80 and 443 (default: 20)')
    
    args = parser.parse_args()
    clean_ports(args.file, args.limit)
