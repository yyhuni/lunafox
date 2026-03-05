"""
Unit tests for Data Generator module.
"""

import pytest
from data_generator import DataGenerator


class TestDataGenerator:
    """Test DataGenerator class."""

    @staticmethod
    def _infer_target_type(name: str) -> str:
        """Infer target type from generated name."""
        if "/" in name:
            return "cidr"
        parts = name.split(".")
        if len(parts) == 4 and all(part.isdigit() for part in parts):
            return "ip"
        return "domain"
    
    def test_generate_organization(self):
        """Test organization generation."""
        org = DataGenerator.generate_organization(0)
        
        assert "name" in org
        assert "description" in org
        assert isinstance(org["name"], str)
        assert isinstance(org["description"], str)
        assert len(org["name"]) > 0
        assert len(org["description"]) > 0
    
    def test_generate_targets(self):
        """Test target generation with correct ratios."""
        targets = DataGenerator.generate_targets(100)
        
        # Count types
        domain_count = sum(1 for t in targets if self._infer_target_type(t["name"]) == "domain")
        ip_count = sum(1 for t in targets if self._infer_target_type(t["name"]) == "ip")
        cidr_count = sum(1 for t in targets if self._infer_target_type(t["name"]) == "cidr")
        
        # Check ratios (allow 5% tolerance)
        assert 65 <= domain_count <= 75  # 70% ± 5%
        assert 15 <= ip_count <= 25      # 20% ± 5%
        assert 5 <= cidr_count <= 15     # 10% ± 5%
        
        # Check all have required fields
        for target in targets:
            assert "name" in target
            assert isinstance(target["name"], str)
            assert len(target["name"]) > 0
    
    def test_generate_domain_target(self):
        """Test domain target format."""
        targets = DataGenerator.generate_targets(10, {"domain": 1.0, "ip": 0, "cidr": 0})
        
        for target in targets:
            assert self._infer_target_type(target["name"]) == "domain"
            assert "." in target["name"]
            assert not target["name"].startswith(".")
            assert not target["name"].endswith(".")
    
    def test_generate_ip_target(self):
        """Test IP target format."""
        targets = DataGenerator.generate_targets(10, {"domain": 0, "ip": 1.0, "cidr": 0})
        
        for target in targets:
            assert self._infer_target_type(target["name"]) == "ip"
            parts = target["name"].split(".")
            assert len(parts) == 4
            for part in parts:
                assert 0 <= int(part) <= 255
    
    def test_generate_cidr_target(self):
        """Test CIDR target format."""
        targets = DataGenerator.generate_targets(10, {"domain": 0, "ip": 0, "cidr": 1.0})
        
        for target in targets:
            assert self._infer_target_type(target["name"]) == "cidr"
            assert "/" in target["name"]
            ip, mask = target["name"].split("/")
            assert int(mask) in [8, 16, 24]
    
    def test_generate_websites(self):
        """Test website generation."""
        target = {"name": "example.com", "type": "domain", "id": 1}
        websites = DataGenerator.generate_websites(target, 5)
        
        assert len(websites) == 5
        
        for website in websites:
            assert "url" in website
            assert "title" in website
            assert "statusCode" in website
            assert "tech" in website
            assert isinstance(website["tech"], list)
            assert website["url"].startswith("http")
            assert "example.com" in website["url"]
    
    def test_generate_subdomains_for_domain(self):
        """Test subdomain generation for domain target."""
        target = {"name": "example.com", "type": "domain", "id": 1}
        subdomains = DataGenerator.generate_subdomains(target, 5)
        
        assert len(subdomains) == 5
        
        for subdomain in subdomains:
            assert isinstance(subdomain, str)
            assert subdomain.endswith("example.com")
            assert subdomain != "example.com"  # Should have prefix
    
    def test_generate_subdomains_for_non_domain(self):
        """Test subdomain generation for non-domain target."""
        target = {"name": "192.168.1.1", "type": "ip", "id": 1}
        subdomains = DataGenerator.generate_subdomains(target, 5)
        
        assert len(subdomains) == 0  # Should return empty for non-domain
    
    def test_generate_endpoints(self):
        """Test endpoint generation."""
        target = {"name": "example.com", "type": "domain", "id": 1}
        endpoints = DataGenerator.generate_endpoints(target, 5)
        
        assert len(endpoints) == 5
        
        for endpoint in endpoints:
            assert "url" in endpoint
            assert "statusCode" in endpoint
            assert "tech" in endpoint
            assert isinstance(endpoint["tech"], list)
    
    def test_generate_directories(self):
        """Test directory generation."""
        target = {"name": "example.com", "type": "domain", "id": 1}
        directories = DataGenerator.generate_directories(target, 5)
        
        assert len(directories) == 5
        
        for directory in directories:
            assert "url" in directory
            assert "status" in directory
            assert "contentLength" in directory
            assert directory["url"].endswith("/")  # Directories end with /
    
    def test_generate_host_ports(self):
        """Test host port generation."""
        target = {"name": "example.com", "type": "domain", "id": 1}
        host_ports = DataGenerator.generate_host_ports(target, 5)
        
        assert len(host_ports) == 5
        
        for hp in host_ports:
            assert "host" in hp
            assert "ip" in hp
            assert "port" in hp
            assert isinstance(hp["port"], int)
            assert 1 <= hp["port"] <= 65535
    
    def test_generate_vulnerabilities(self):
        """Test vulnerability generation."""
        target = {"name": "example.com", "type": "domain", "id": 1}
        vulns = DataGenerator.generate_vulnerabilities(target, 5)
        
        assert len(vulns) == 5
        
        for vuln in vulns:
            assert "url" in vuln
            assert "vulnType" in vuln
            assert "severity" in vuln
            assert "cvssScore" in vuln
            assert vuln["severity"] in ["critical", "high", "medium", "low", "info"]
            assert 0 <= vuln["cvssScore"] <= 10

    def test_generate_scan_uses_workflow_ids(self):
        """Test scan payload uses workflowIds only."""
        target = {"name": "example.com", "type": "domain", "id": 1}
        scan = DataGenerator.generate_scan(target)

        assert scan["targetId"] == 1
        assert "workflowIds" in scan
        assert isinstance(scan["workflowIds"], list)
        assert 1 <= len(scan["workflowIds"]) <= 3
        assert "workflowNames" not in scan


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
