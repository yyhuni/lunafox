"""
Data Generator Module

Generates random but reasonable test data.
"""

import random
from typing import List, Dict, Any


class DataGenerator:
    """Generates random test data for seeding."""
    
    # Organization data templates
    ORG_NAMES = [
        "Acme Corporation", "TechStart Labs", "Global Finance", "HealthCare Plus",
        "E-Commerce Platform", "Smart City Systems", "Educational Tech", "Green Energy",
        "CyberSec Defense", "CloudNative Systems", "DataFlow Analytics", "MobileFirst Tech",
        "Quantum Research", "Autonomous Vehicles", "Biotech Innovations", "Space Technology",
        "AI Research Lab", "Blockchain Solutions", "IoT Platform", "DevOps Enterprise",
        "Security Operations", "Data Science Hub", "Machine Learning Co", "Network Solutions",
        "Infrastructure Corp", "Platform Services", "Digital Transformation", "Innovation Hub",
        "Tech Consulting", "Software Factory",
    ]
    
    DIVISIONS = [
        "Global", "Asia Pacific", "EMEA", "Americas", "R&D", "Cloud Services",
        "Security Team", "Innovation Lab", "Enterprise", "Consumer Products",
    ]
    
    DESCRIPTIONS = [
        "A leading technology company specializing in enterprise software solutions and cloud computing services.",
        "Innovative research lab focused on artificial intelligence and machine learning applications.",
        "Global financial services provider offering digital banking and payment solutions.",
        "Healthcare technology company developing electronic health records and telemedicine platforms.",
        "E-commerce platform serving millions of customers with B2B and B2C solutions.",
        "Smart city infrastructure provider specializing in IoT and urban management systems.",
        "Educational technology company providing online learning platforms and courses.",
        "Renewable energy management company focused on solar and wind power optimization.",
        "Cybersecurity firm offering penetration testing and security consulting services.",
        "Cloud-native systems developer specializing in Kubernetes and microservices.",
    ]
    
    @staticmethod
    def generate_organization(index: int) -> Dict[str, Any]:
        """
        Generate organization data.
        
        Args:
            index: Organization index (for uniqueness)
            
        Returns:
            Organization data dictionary with camelCase fields
        """
        suffix = random.randint(1000, 9999)
        name = f"{DataGenerator.ORG_NAMES[index % len(DataGenerator.ORG_NAMES)]} - {random.choice(DataGenerator.DIVISIONS)} ({suffix}-{index})"
        description = random.choice(DataGenerator.DESCRIPTIONS)
        
        return {
            "name": name,
            "description": description
        }

    
    # Target data templates
    ENVS = ["prod", "staging", "dev", "test", "api", "app", "www", "admin", "portal", "dashboard"]
    COMPANIES = ["acme", "techstart", "globalfinance", "healthcare", "ecommerce", "smartcity", "cybersec", "cloudnative", "dataflow", "mobilefirst"]
    TLDS = [".com", ".io", ".net", ".org", ".dev", ".app", ".cloud", ".tech"]
    
    @staticmethod
    def generate_targets(count: int, target_type_ratios: Dict[str, float] = None) -> List[Dict[str, Any]]:
        """
        Generate target data with specified type ratios.
        
        Args:
            count: Total number of targets to generate
            target_type_ratios: Type distribution (default: domain 70%, ip 20%, cidr 10%)
            
        Returns:
            List of target data dictionaries with camelCase fields
        """
        if target_type_ratios is None:
            target_type_ratios = {"domain": 0.7, "ip": 0.2, "cidr": 0.1}
        
        targets = []
        suffix = random.randint(1000, 9999)
        used_names = set()
        
        # Generate domains
        domain_count = int(count * target_type_ratios.get("domain", 0.7))
        for i in range(domain_count):
            while True:
                env = random.choice(DataGenerator.ENVS)
                company = random.choice(DataGenerator.COMPANIES)
                tld = random.choice(DataGenerator.TLDS)
                name = f"{env}.{company}-{suffix + i}{tld}"
                
                if name not in used_names:
                    used_names.add(name)
                    targets.append({"name": name})
                    break
        
        # Generate IPs
        ip_count = int(count * target_type_ratios.get("ip", 0.2))
        for i in range(ip_count):
            while True:
                name = f"{random.randint(1, 223)}.{random.randint(0, 255)}.{random.randint(0, 255)}.{random.randint(1, 254)}"
                
                if name not in used_names:
                    used_names.add(name)
                    targets.append({"name": name})
                    break
        
        # Generate CIDRs
        cidr_count = count - len(targets)  # Remaining
        for i in range(cidr_count):
            while True:
                mask = random.choice([8, 16, 24])
                name = f"{random.randint(1, 223)}.{random.randint(0, 255)}.{random.randint(0, 255)}.0/{mask}"
                
                if name not in used_names:
                    used_names.add(name)
                    targets.append({"name": name})
                    break
        
        return targets

    
    # Website data templates
    PROTOCOLS = ["https://", "http://"]
    SUBDOMAINS = ["www", "api", "app", "admin", "portal", "dashboard", "dev", "staging", "test", "cdn", "static", "assets"]
    PATHS = ["", "/", "/api", "/v1", "/v2", "/login", "/dashboard", "/admin", "/app", "/docs"]
    PORTS = ["", ":8080", ":8443", ":3000", ":443"]
    
    TITLES = [
        "Welcome - Dashboard", "Admin Panel", "API Documentation", "Login Portal",
        "Home Page", "User Dashboard", "Settings", "Analytics", "Reports",
        "Management Console", "Control Panel", "Service Status", "Developer Portal",
    ]
    
    WEBSERVERS = [
        "nginx/1.24.0", "Apache/2.4.57", "cloudflare", "Microsoft-IIS/10.0",
        "nginx", "Apache", "LiteSpeed", "Caddy", "Traefik",
    ]
    
    CONTENT_TYPES = [
        "text/html; charset=utf-8", "text/html", "application/json",
        "text/html; charset=UTF-8", "application/xhtml+xml",
    ]
    
    TECH_STACKS = [
        ["nginx", "PHP", "MySQL"],
        ["Apache", "Python", "PostgreSQL"],
        ["nginx", "Node.js", "MongoDB"],
        ["cloudflare", "React", "GraphQL"],
        ["nginx", "Vue.js", "Redis"],
        ["Apache", "Java", "Oracle"],
        ["nginx", "Go", "PostgreSQL"],
        ["cloudflare", "Next.js", "Vercel"],
    ]
    
    STATUS_CODES = [200, 200, 200, 200, 200, 301, 302, 403, 404, 500]
    
    @staticmethod
    def generate_websites(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate website data for a target.

        Args:
            target: Target data (must have 'name' and 'type')
            count: Number of websites to generate

        Returns:
            List of website data dictionaries with camelCase fields
        """
        websites = []
        used_urls = set()

        # Calculate total possible combinations to avoid infinite loop
        max_combinations = len(DataGenerator.PROTOCOLS) * len(DataGenerator.SUBDOMAINS) * len(DataGenerator.PORTS) * len(DataGenerator.PATHS)
        actual_count = min(count, max_combinations)

        i = 0
        attempts = 0
        max_attempts = actual_count * 10  # Prevent infinite loop

        while len(websites) < actual_count and attempts < max_attempts:
            attempts += 1

            # Use different strategies to generate unique combinations
            protocol_idx = i % len(DataGenerator.PROTOCOLS)
            subdomain_idx = (i // len(DataGenerator.PROTOCOLS)) % len(DataGenerator.SUBDOMAINS)
            port_idx = (i // (len(DataGenerator.PROTOCOLS) * len(DataGenerator.SUBDOMAINS))) % len(DataGenerator.PORTS)
            path_idx = (i // (len(DataGenerator.PROTOCOLS) * len(DataGenerator.SUBDOMAINS) * len(DataGenerator.PORTS))) % len(DataGenerator.PATHS)

            protocol = DataGenerator.PROTOCOLS[protocol_idx]
            subdomain = DataGenerator.SUBDOMAINS[subdomain_idx]
            port = DataGenerator.PORTS[port_idx]
            path = DataGenerator.PATHS[path_idx]

            # Generate URL based on target type
            if target["type"] == "domain":
                url = f"{protocol}{subdomain}.{target['name']}{port}{path}"
            elif target["type"] == "ip":
                url = f"{protocol}{target['name']}{port}{path}"
            elif target["type"] == "cidr":
                # Use base IP from CIDR
                base_ip = target["name"].split("/")[0]
                url = f"{protocol}{base_ip}{port}{path}"
            else:
                i += 1
                continue

            # Skip if URL already exists
            if url in used_urls:
                i += 1
                continue

            used_urls.add(url)

            status_code = DataGenerator.STATUS_CODES[len(websites) % len(DataGenerator.STATUS_CODES)]
            content_length = 1000 + (len(websites) * 100)
            tech = DataGenerator.TECH_STACKS[len(websites) % len(DataGenerator.TECH_STACKS)]
            vhost = (len(websites) % 5 == 0)  # 20% are vhost

            websites.append({
                "url": url,
                "title": DataGenerator.TITLES[len(websites) % len(DataGenerator.TITLES)],
                "statusCode": status_code,
                "contentLength": content_length,
                "contentType": DataGenerator.CONTENT_TYPES[len(websites) % len(DataGenerator.CONTENT_TYPES)],
                "webserver": DataGenerator.WEBSERVERS[len(websites) % len(DataGenerator.WEBSERVERS)],
                "tech": tech,
                "vhost": vhost,
            })

            i += 1

        return websites

    
    # Subdomain prefixes
    SUBDOMAIN_PREFIXES = [
        "www", "api", "app", "admin", "portal", "dashboard", "dev", "staging",
        "test", "cdn", "static", "assets", "mail", "blog", "docs", "support",
        "auth", "login", "shop", "store",
    ]
    
    @staticmethod
    def generate_subdomains(target: Dict[str, Any], count: int) -> List[str]:
        """
        Generate subdomain data for a domain target.

        Args:
            target: Target data (must be type 'domain')
            count: Number of subdomains to generate

        Returns:
            List of subdomain names (strings)
            Empty list if target is not a domain
        """
        if target["type"] != "domain":
            return []

        subdomains = []
        target_name = target['name']

        # Generate subdomains that end with .target_name
        # e.g., target=portal.example.com -> www.portal.example.com
        for i in range(count):
            prefix = DataGenerator.SUBDOMAIN_PREFIXES[i % len(DataGenerator.SUBDOMAIN_PREFIXES)]
            name = f"{prefix}.{target_name}"

            # Skip if same as target name
            if name == target_name:
                continue

            subdomains.append(name)

        return subdomains
        
        return subdomains

    
    # Endpoint data templates
    API_PATHS = [
        "/api/v1/users", "/api/v1/products", "/api/v2/orders", "/login", "/dashboard",
        "/admin/settings", "/app/config", "/docs/api", "/health", "/metrics",
        "/api/auth/login", "/api/auth/logout", "/api/data/export", "/api/search",
        "/graphql", "/ws/connect", "/api/upload", "/api/download", "/status", "/version",
    ]
    
    ENDPOINT_TITLES = [
        "API Endpoint", "User Service", "Product API", "Authentication",
        "Dashboard API", "Admin Panel", "Configuration", "Documentation",
        "Health Check", "Metrics Endpoint", "GraphQL API", "WebSocket",
    ]
    
    API_TECH_STACKS = [
        ["nginx", "Node.js", "Express"],
        ["Apache", "Python", "FastAPI"],
        ["nginx", "Go", "Gin"],
        ["cloudflare", "Rust", "Actix"],
    ]
    
    @staticmethod
    def generate_endpoints(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate endpoint data for a target.

        Args:
            target: Target data (must have 'name' and 'type')
            count: Number of endpoints to generate

        Returns:
            List of endpoint data dictionaries with camelCase fields
        """
        endpoints = []
        used_urls = set()

        # Calculate total possible combinations
        max_combinations = len(DataGenerator.PROTOCOLS) * len(DataGenerator.SUBDOMAINS) * len(DataGenerator.API_PATHS)
        actual_count = min(count, max_combinations)

        i = 0
        attempts = 0
        max_attempts = actual_count * 10

        while len(endpoints) < actual_count and attempts < max_attempts:
            attempts += 1

            protocol_idx = i % len(DataGenerator.PROTOCOLS)
            subdomain_idx = (i // len(DataGenerator.PROTOCOLS)) % len(DataGenerator.SUBDOMAINS)
            path_idx = (i // (len(DataGenerator.PROTOCOLS) * len(DataGenerator.SUBDOMAINS))) % len(DataGenerator.API_PATHS)

            protocol = DataGenerator.PROTOCOLS[protocol_idx]
            subdomain = DataGenerator.SUBDOMAINS[subdomain_idx]
            path = DataGenerator.API_PATHS[path_idx]

            # Generate URL based on target type
            if target["type"] == "domain":
                url = f"{protocol}{subdomain}.{target['name']}{path}"
            elif target["type"] == "ip":
                url = f"{protocol}{target['name']}{path}"
            elif target["type"] == "cidr":
                base_ip = target["name"].split("/")[0]
                url = f"{protocol}{base_ip}{path}"
            else:
                i += 1
                continue

            # Skip if URL already exists
            if url in used_urls:
                i += 1
                continue

            used_urls.add(url)

            status_code = DataGenerator.STATUS_CODES[len(endpoints) % len(DataGenerator.STATUS_CODES)]
            content_length = 500 + (len(endpoints) * 50)
            tech = DataGenerator.API_TECH_STACKS[len(endpoints) % len(DataGenerator.API_TECH_STACKS)]
            vhost = (len(endpoints) % 10 == 0)  # 10% are vhost

            endpoints.append({
                "url": url,
                "title": DataGenerator.ENDPOINT_TITLES[len(endpoints) % len(DataGenerator.ENDPOINT_TITLES)],
                "statusCode": status_code,
                "contentLength": content_length,
                "contentType": "application/json",
                "webserver": DataGenerator.WEBSERVERS[len(endpoints) % len(DataGenerator.WEBSERVERS)],
                "tech": tech,
                "vhost": vhost,
            })

            i += 1

        return endpoints

    
    # Directory paths
    DIRECTORIES = [
        "/admin/", "/backup/", "/config/", "/data/", "/debug/",
        "/files/", "/images/", "/js/", "/css/", "/uploads/",
        "/api/", "/docs/", "/logs/", "/temp/", "/cache/",
        "/static/", "/assets/", "/media/", "/public/", "/private/",
    ]
    
    DIR_STATUS_CODES = [200, 200, 200, 301, 302, 403, 404]
    
    @staticmethod
    def generate_directories(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate directory data for a target.

        Args:
            target: Target data (must have 'name' and 'type')
            count: Number of directories to generate

        Returns:
            List of directory data dictionaries with camelCase fields
        """
        directories = []
        used_urls = set()

        # Calculate total possible combinations
        max_combinations = len(DataGenerator.PROTOCOLS) * len(DataGenerator.SUBDOMAINS) * len(DataGenerator.DIRECTORIES)
        actual_count = min(count, max_combinations)

        i = 0
        attempts = 0
        max_attempts = actual_count * 10

        while len(directories) < actual_count and attempts < max_attempts:
            attempts += 1

            protocol_idx = i % len(DataGenerator.PROTOCOLS)
            subdomain_idx = (i // len(DataGenerator.PROTOCOLS)) % len(DataGenerator.SUBDOMAINS)
            dir_idx = (i // (len(DataGenerator.PROTOCOLS) * len(DataGenerator.SUBDOMAINS))) % len(DataGenerator.DIRECTORIES)

            protocol = DataGenerator.PROTOCOLS[protocol_idx]
            subdomain = DataGenerator.SUBDOMAINS[subdomain_idx]
            dir_path = DataGenerator.DIRECTORIES[dir_idx]

            # Generate URL based on target type
            if target["type"] == "domain":
                url = f"{protocol}{subdomain}.{target['name']}{dir_path}"
            elif target["type"] == "ip":
                url = f"{protocol}{target['name']}{dir_path}"
            elif target["type"] == "cidr":
                base_ip = target["name"].split("/")[0]
                url = f"{protocol}{base_ip}{dir_path}"
            else:
                i += 1
                continue

            # Skip if URL already exists
            if url in used_urls:
                i += 1
                continue

            used_urls.add(url)

            status = DataGenerator.DIR_STATUS_CODES[len(directories) % len(DataGenerator.DIR_STATUS_CODES)]
            content_length = 1000 + (len(directories) * 100)
            duration = 50 + (len(directories) * 5)

            directories.append({
                "url": url,
                "status": status,
                "contentLength": content_length,
                "contentType": DataGenerator.CONTENT_TYPES[len(directories) % len(DataGenerator.CONTENT_TYPES)],
                "duration": duration,
            })

            i += 1

        return directories

    
    # Common ports
    COMMON_PORTS = [22, 80, 443, 8080, 8443, 3000, 3306, 5432, 6379, 27017, 9200, 9300, 5000, 8000, 8080, 9000, 9090, 10000]
    
    @staticmethod
    def generate_host_ports(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate host port mapping data for a target.
        
        Args:
            target: Target data (must have 'name' and 'type')
            count: Number of host port mappings to generate
            
        Returns:
            List of host port data dictionaries with camelCase fields
        """
        host_ports = []
        
        # Generate base IP for this target
        base_ip1 = random.randint(1, 223)
        base_ip2 = random.randint(0, 255)
        base_ip3 = random.randint(0, 255)
        
        for i in range(count):
            # Generate IP
            ip = f"{base_ip1}.{base_ip2}.{base_ip3}.{(i % 254) + 1}"
            
            # Generate host based on target type
            # For domain targets, use base domain without subdomain prefix
            if target["type"] == "domain":
                target_name = target["name"]
                parts = target_name.split('.')
                if len(parts) > 2:
                    # Has subdomain, use base domain
                    host = '.'.join(parts[1:])
                else:
                    # No subdomain, use as is
                    host = target_name
            elif target["type"] == "ip":
                host = target["name"]
            elif target["type"] == "cidr":
                host = target["name"].split("/")[0]
            else:
                continue
            
            port = DataGenerator.COMMON_PORTS[i % len(DataGenerator.COMMON_PORTS)]
            
            host_ports.append({
                "host": host,
                "ip": ip,
                "port": port,
            })
        
        return host_ports

    
    # Vulnerability data templates
    VULN_TYPES = [
        "SQL Injection", "Cross-Site Scripting (XSS)", "Remote Code Execution",
        "Server-Side Request Forgery (SSRF)", "Local File Inclusion (LFI)",
        "XML External Entity (XXE)", "Insecure Deserialization", "Command Injection",
        "Path Traversal", "Open Redirect", "CRLF Injection", "CORS Misconfiguration",
        "Information Disclosure", "Authentication Bypass", "Privilege Escalation",
    ]
    
    SEVERITIES = ["critical", "high", "high", "medium", "medium", "medium", "low", "low", "info"]
    
    SOURCES = ["nuclei", "dalfox", "sqlmap", "burpsuite", "manual"]
    
    DESCRIPTIONS = [
        "A SQL injection vulnerability was found that allows an attacker to execute arbitrary SQL queries.",
        "A reflected XSS vulnerability exists that could allow attackers to inject malicious scripts.",
        "Remote code execution is possible through unsafe deserialization of user input.",
        "SSRF vulnerability allows an attacker to make requests to internal services.",
        "Local file inclusion vulnerability could expose sensitive files on the server.",
        "XXE vulnerability found that could lead to information disclosure or SSRF.",
        "Insecure deserialization could lead to remote code execution.",
        "OS command injection vulnerability found in user-controlled input.",
        "Path traversal vulnerability allows access to files outside the web root.",
        "Open redirect vulnerability could be used for phishing attacks.",
    ]
    
    VULN_PATHS = [
        "/login", "/api/v1/users", "/api/v1/search", "/admin/config",
        "/api/export", "/upload", "/api/v2/data", "/graphql",
        "/api/auth", "/dashboard", "/api/profile", "/settings",
    ]
    
    CVSS_SCORES = [9.8, 9.1, 8.6, 7.5, 6.5, 5.4, 4.3, 3.1, 2.0]
    
    @staticmethod
    def generate_vulnerabilities(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate vulnerability data for a target.

        Args:
            target: Target data (must have 'name' and 'type')
            count: Number of vulnerabilities to generate

        Returns:
            List of vulnerability data dictionaries with camelCase fields
        """
        vulnerabilities = []

        for i in range(count):
            path = DataGenerator.VULN_PATHS[i % len(DataGenerator.VULN_PATHS)]

            # Generate URL based on target type
            if target["type"] == "domain":
                url = f"https://www.{target['name']}{path}"
            elif target["type"] == "ip":
                url = f"https://{target['name']}{path}"
            elif target["type"] == "cidr":
                base_ip = target["name"].split("/")[0]
                url = f"https://{base_ip}{path}"
            else:
                continue

            cvss_score = DataGenerator.CVSS_SCORES[i % len(DataGenerator.CVSS_SCORES)]

            vulnerabilities.append({
                "url": url,
                "vulnType": DataGenerator.VULN_TYPES[i % len(DataGenerator.VULN_TYPES)],
                "severity": DataGenerator.SEVERITIES[i % len(DataGenerator.SEVERITIES)],
                "source": DataGenerator.SOURCES[i % len(DataGenerator.SOURCES)],
                "cvssScore": cvss_score,
                "description": DataGenerator.DESCRIPTIONS[i % len(DataGenerator.DESCRIPTIONS)],
            })

        return vulnerabilities

    # Screenshot data templates
    SCREENSHOT_STATUS_CODES = [200, 200, 200, 200, 301, 302, 403, 404]

    @staticmethod
    def generate_screenshots(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate screenshot data for a target.

        Args:
            target: Target data (must have 'name' and 'type')
            count: Number of screenshots to generate

        Returns:
            List of screenshot data dictionaries with camelCase fields
        """
        screenshots = []
        used_urls = set()

        max_combinations = (
            len(DataGenerator.PROTOCOLS) *
            len(DataGenerator.SUBDOMAINS) *
            len(DataGenerator.PORTS)
        )
        actual_count = min(count, max_combinations)

        i = 0
        attempts = 0
        max_attempts = actual_count * 10

        while len(screenshots) < actual_count and attempts < max_attempts:
            attempts += 1

            protocol_idx = i % len(DataGenerator.PROTOCOLS)
            subdomain_idx = (i // len(DataGenerator.PROTOCOLS)) % len(DataGenerator.SUBDOMAINS)
            port_idx = (
                i // (len(DataGenerator.PROTOCOLS) * len(DataGenerator.SUBDOMAINS))
            ) % len(DataGenerator.PORTS)

            protocol = DataGenerator.PROTOCOLS[protocol_idx]
            subdomain = DataGenerator.SUBDOMAINS[subdomain_idx]
            port = DataGenerator.PORTS[port_idx]

            # Generate URL based on target type
            if target["type"] == "domain":
                url = f"{protocol}{subdomain}.{target['name']}{port}"
            elif target["type"] == "ip":
                url = f"{protocol}{target['name']}{port}"
            elif target["type"] == "cidr":
                base_ip = target["name"].split("/")[0]
                url = f"{protocol}{base_ip}{port}"
            else:
                i += 1
                continue

            if url in used_urls:
                i += 1
                continue

            used_urls.add(url)

            status_code = DataGenerator.SCREENSHOT_STATUS_CODES[
                len(screenshots) % len(DataGenerator.SCREENSHOT_STATUS_CODES)
            ]

            screenshots.append({
                "url": url,
                "statusCode": status_code,
                # Note: image field is optional, omitted for seed data
            })

            i += 1

        return screenshots

    # Scan data templates
    ENGINE_IDS = [1, 2, 3, 4, 5]
    ENGINE_NAMES = [
        "subdomain_discovery",
        "port_scan",
        "web_crawl",
        "vulnerability_scan",
        "screenshot_capture",
    ]

    SCAN_CONFIGURATIONS = [
        "subdomain_discovery:\\n  passive_tools:\\n    subfinder:\\n      enabled: true",
        "port_scan:\\n  nmap:\\n    enabled: true\\n    ports: top-1000",
        "web_crawl:\\n  httpx:\\n    enabled: true",
        "vulnerability_scan:\\n  nuclei:\\n    enabled: true",
    ]

    @staticmethod
    def generate_scan(target: Dict[str, Any]) -> Dict[str, Any]:
        """
        Generate scan data for a target.

        Args:
            target: Target data (must have 'id')

        Returns:
            Scan initiate request data
        """
        engine_count = random.randint(1, 3)
        engine_indices = random.sample(range(len(DataGenerator.ENGINE_IDS)), engine_count)

        return {
            "targetId": target["id"],
            "engineIds": [DataGenerator.ENGINE_IDS[i] for i in engine_indices],
            "engineNames": [DataGenerator.ENGINE_NAMES[i] for i in engine_indices],
            "configuration": random.choice(DataGenerator.SCAN_CONFIGURATIONS),
        }

    @staticmethod
    def generate_subdomain_snapshots(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate subdomain snapshot data for a domain target.

        Args:
            target: Target data (must be type 'domain')
            count: Number of subdomains to generate

        Returns:
            List of subdomain snapshot items
        """
        if target["type"] != "domain":
            return []

        subdomains = []
        target_name = target['name']

        parts = target_name.split('.')
        if len(parts) > 2:
            base_domain = '.'.join(parts[1:])
        else:
            base_domain = target_name

        for i in range(count):
            prefix = DataGenerator.SUBDOMAIN_PREFIXES[i % len(DataGenerator.SUBDOMAIN_PREFIXES)]
            name = f"{prefix}.{base_domain}"

            if name == target_name:
                continue

            subdomains.append({"name": name})

        return subdomains

    @staticmethod
    def generate_host_port_snapshots(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate host port snapshot data for a target.

        Args:
            target: Target data
            count: Number of host port mappings to generate

        Returns:
            List of host port snapshot items
        """
        host_ports = []

        base_ip1 = random.randint(1, 223)
        base_ip2 = random.randint(0, 255)
        base_ip3 = random.randint(0, 255)

        for i in range(count):
            ip = f"{base_ip1}.{base_ip2}.{base_ip3}.{(i % 254) + 1}"

            if target["type"] == "domain":
                target_name = target["name"]
                parts = target_name.split('.')
                if len(parts) > 2:
                    host = '.'.join(parts[1:])
                else:
                    host = target_name
            elif target["type"] == "ip":
                host = target["name"]
            elif target["type"] == "cidr":
                host = target["name"].split("/")[0]
            else:
                continue

            port = DataGenerator.COMMON_PORTS[i % len(DataGenerator.COMMON_PORTS)]

            host_ports.append({
                "host": host,
                "ip": ip,
                "port": port,
            })

        return host_ports

    @staticmethod
    def generate_vulnerability_snapshots(target: Dict[str, Any], count: int) -> List[Dict[str, Any]]:
        """
        Generate vulnerability snapshot data for a target.

        Args:
            target: Target data
            count: Number of vulnerabilities to generate

        Returns:
            List of vulnerability snapshot items with required severity field
        """
        vulnerabilities = []

        for i in range(count):
            path = DataGenerator.VULN_PATHS[i % len(DataGenerator.VULN_PATHS)]

            if target["type"] == "domain":
                url = f"https://www.{target['name']}{path}"
            elif target["type"] == "ip":
                url = f"https://{target['name']}{path}"
            elif target["type"] == "cidr":
                base_ip = target["name"].split("/")[0]
                url = f"https://{base_ip}{path}"
            else:
                continue

            cvss_score = DataGenerator.CVSS_SCORES[i % len(DataGenerator.CVSS_SCORES)]

            vulnerabilities.append({
                "url": url,
                "vulnType": DataGenerator.VULN_TYPES[i % len(DataGenerator.VULN_TYPES)],
                "severity": DataGenerator.SEVERITIES[i % len(DataGenerator.SEVERITIES)],
                "source": DataGenerator.SOURCES[i % len(DataGenerator.SOURCES)],
                "cvssScore": cvss_score,
                "description": DataGenerator.DESCRIPTIONS[i % len(DataGenerator.DESCRIPTIONS)],
            })

        return vulnerabilities
