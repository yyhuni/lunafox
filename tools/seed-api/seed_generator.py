#!/usr/bin/env python3
"""
API-Based Seed Data Generator

Main entry point for generating test data via API calls.
"""

import argparse
import sys


def check_requirements():
    """Check Python version and dependencies."""
    # Check Python version
    if sys.version_info < (3, 8):
        print("❌ Error: Python 3.8+ is required")
        print(f"   Current version: {sys.version}")
        sys.exit(1)
    
    # Check requests library
    try:
        import requests
    except ImportError:
        print("❌ Error: requests library is not installed")
        print("   Please run: pip install -r requirements.txt")
        sys.exit(1)


def parse_args():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Generate test data via API calls",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Generate default data (15 orgs, 15 targets per org)
  python seed_generator.py
  
  # Generate small dataset
  python seed_generator.py --orgs 5 --targets-per-org 10
  
  # Clear existing data first
  python seed_generator.py --clear
  
  # Use custom API URL
  python seed_generator.py --api-url http://192.168.1.100:8080
        """
    )
    
    parser.add_argument(
        "--api-url",
        default="http://localhost:8080",
        help="API base URL (default: http://localhost:8080)"
    )
    
    parser.add_argument(
        "--username",
        default="admin",
        help="Username for authentication (default: admin)"
    )
    
    parser.add_argument(
        "--password",
        default="admin",
        help="Password for authentication (default: admin)"
    )
    
    parser.add_argument(
        "--orgs",
        type=int,
        default=15,
        help="Number of organizations to generate (default: 15)"
    )
    
    parser.add_argument(
        "--targets-per-org",
        type=int,
        default=15,
        help="Number of targets per organization (default: 15)"
    )
    
    parser.add_argument(
        "--assets-per-target",
        type=int,
        default=15,
        help="Number of assets per target (default: 15)"
    )
    
    parser.add_argument(
        "--clear",
        action="store_true",
        help="Clear existing data before generating"
    )
    
    parser.add_argument(
        "--batch-size",
        type=int,
        default=100,
        help="Batch size for bulk operations (default: 100)"
    )
    
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Show verbose output"
    )
    
    return parser.parse_args()


def main():
    """Main entry point."""
    # Check requirements first
    check_requirements()
    
    args = parse_args()
    
    from api_client import APIClient, APIError
    from progress import ProgressTracker
    from error_handler import ErrorHandler
    
    # Initialize components
    client = APIClient(args.api_url, args.username, args.password)
    progress = ProgressTracker()
    error_handler = ErrorHandler()
    
    try:
        # Login
        print("🔐 Logging in...")
        client.login()
        print("   ✓ Authenticated")
        print()
        
        # Clear data if requested
        if args.clear:
            clear_data(client, progress)
        
        # Calculate counts
        target_count = args.orgs * args.targets_per_org
        
        print("🚀 Starting test data generation...")
        print(f"   Organizations: {args.orgs}")
        print(f"   Targets: {target_count} ({args.targets_per_org} per org)")
        print(f"   Assets per target: {args.assets_per_target}")
        print()
        
        # Create organizations
        org_ids = create_organizations(client, progress, error_handler, args.orgs)
        
        # Create targets
        targets = create_targets(client, progress, error_handler, target_count)
        target_ids = [t["id"] for t in targets]
        
        # Link targets to organizations
        link_targets_to_organizations(client, progress, error_handler, target_ids, org_ids)
        
        # Create assets
        create_assets(client, progress, error_handler, targets, args.assets_per_target, args.batch_size)

        # Create scans and snapshots
        scan_ids = create_scans(client, progress, error_handler, targets)
        if scan_ids:
            create_snapshots(client, progress, error_handler, targets, scan_ids, args.assets_per_target, args.batch_size)

        # Print summary
        progress.print_summary()
        
    except APIError as e:
        print(f"\n❌ API Error: {e}")
        if args.verbose and e.response_data:
            print(f"   Response: {e.response_data}")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n\n⚠ Interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Unexpected error: {e}")
        if args.verbose:
            import traceback
            traceback.print_exc()
        sys.exit(1)


def clear_data(client, progress):
    """
    Clear existing data.
    
    Args:
        client: API client
        progress: Progress tracker
    """
    print("🗑️  Clearing existing data...")
    
    # Delete in correct order (child tables first)
    delete_operations = [
        ("scans", "/api/scans/bulk-delete"),
        ("screenshots", "/api/screenshots/bulk-delete"),
        ("vulnerabilities", "/api/vulnerabilities/bulk-delete"),
        ("host ports", "/api/host-ports/bulk-delete"),
        ("directories", "/api/directories/bulk-delete"),
        ("endpoints", "/api/endpoints/bulk-delete"),
        ("subdomains", "/api/subdomains/bulk-delete"),
        ("websites", "/api/websites/bulk-delete"),
        ("targets", "/api/targets/bulk-delete"),
        ("organizations", "/api/organizations/bulk-delete"),
    ]
    
    for name, endpoint in delete_operations:
        try:
            # Get all IDs
            list_endpoint = endpoint.replace("/bulk-delete", "")
            response = client.get(list_endpoint, {"page": 1, "pageSize": 10000})
            
            if "results" in response and len(response["results"]) > 0:
                ids = [item["id"] for item in response["results"]]
                
                # Delete in batches
                batch_size = 100
                for i in range(0, len(ids), batch_size):
                    batch_ids = ids[i:i + batch_size]
                    client.post(endpoint, {"ids": batch_ids})
                
                print(f"   ✓ Deleted {len(ids)} {name}")
            else:
                print(f"   ✓ No {name} to delete")
                
        except Exception as e:
            print(f"   ⚠ Failed to delete {name}: {e}")
    
    print("   ✓ Data cleared")
    print()


def create_organizations(client, progress, error_handler, count):
    """
    Create organizations.
    
    Args:
        client: API client
        progress: Progress tracker
        error_handler: Error handler
        count: Number of organizations to create
        
    Returns:
        List of organization IDs
    """
    from data_generator import DataGenerator
    
    progress.start_phase(f"Creating {count} organizations", count, "🏢")
    
    org_ids = []
    
    for i in range(count):
        try:
            org_data = DataGenerator.generate_organization(i)
            
            result = error_handler.retry_with_backoff(
                client.post,
                "/api/organizations",
                org_data
            )
            
            org_ids.append(result["id"])
            progress.add_success(1)
            progress.update(i + 1)
            
        except Exception as e:
            progress.add_error(str(e))
            error_handler.log_error(str(e), org_data)
    
    progress.finish_phase()
    return org_ids


def create_targets(client, progress, error_handler, count):
    """
    Create targets in batches.
    
    Args:
        client: API client
        progress: Progress tracker
        error_handler: Error handler
        count: Number of targets to create
        
    Returns:
        List of target dictionaries (with id, name, type)
    """
    from data_generator import DataGenerator
    
    progress.start_phase(f"Creating {count} targets", count, "🎯")
    
    # Generate all targets
    targets_data = DataGenerator.generate_targets(count)
    target_names = [t["name"] for t in targets_data]
    
    # Batch create (100 per batch)
    batch_size = 100
    total_created = 0
    
    for i in range(0, len(targets_data), batch_size):
        batch = targets_data[i:i + batch_size]
        
        try:
            result = error_handler.retry_with_backoff(
                client.post,
                "/api/targets/bulk-create",
                {"targets": batch}
            )
            
            created_count = result.get("createdCount", 0)
            total_created += created_count
            
            progress.add_success(created_count)
            progress.update(min(i + batch_size, len(targets_data)))
            
        except Exception as e:
            progress.add_error(str(e))
            error_handler.log_error(str(e), {"targets": batch})
    
    progress.finish_phase()
    
    # Fetch created targets by names to get IDs
    all_targets = []
    if total_created > 0:
        try:
            response = client.get("/api/targets", {"page": 1, "pageSize": count})
            if "results" in response:
                # Filter targets that match our generated names
                all_targets = [t for t in response["results"] if t["name"] in target_names]
        except Exception as e:
            error_handler.log_error(str(e), {"action": "fetch_created_targets"})
    
    return all_targets


def link_targets_to_organizations(client, progress, error_handler, target_ids, org_ids):
    """
    Link targets to organizations evenly.
    
    Args:
        client: API client
        progress: Progress tracker
        error_handler: Error handler
        target_ids: List of target IDs
        org_ids: List of organization IDs
    """
    if not org_ids or not target_ids:
        return
    
    progress.start_phase(f"Linking {len(target_ids)} targets to organizations", len(target_ids), "🔗")
    
    # Distribute targets evenly across organizations
    targets_per_org = len(target_ids) // len(org_ids)
    
    for org_idx, org_id in enumerate(org_ids):
        start_idx = org_idx * targets_per_org
        end_idx = start_idx + targets_per_org
        
        # Last org gets remaining targets
        if org_idx == len(org_ids) - 1:
            end_idx = len(target_ids)
        
        org_target_ids = target_ids[start_idx:end_idx]
        
        if not org_target_ids:
            continue
        
        try:
            error_handler.retry_with_backoff(
                client.post,
                f"/api/organizations/{org_id}/link_targets",
                {"targetIds": org_target_ids}
            )
            
            progress.add_success(len(org_target_ids))
            progress.update(end_idx)
            
        except Exception as e:
            progress.add_error(str(e))
            error_handler.log_error(str(e), {"orgId": org_id, "targetIds": org_target_ids})
    
    progress.finish_phase()



def create_assets(client, progress, error_handler, targets, assets_per_target, batch_size):
    """
    Create all assets for targets.
    
    Args:
        client: API client
        progress: Progress tracker
        error_handler: Error handler
        targets: List of target dictionaries
        assets_per_target: Number of assets per target
        batch_size: Batch size for bulk operations
    """
    from data_generator import DataGenerator
    
    # Create websites
    progress.start_phase(f"Creating websites", len(targets) * assets_per_target, "🌐")
    
    for target in targets:
        websites = DataGenerator.generate_websites(target, assets_per_target)
        
        # Batch create
        for i in range(0, len(websites), batch_size):
            batch = websites[i:i + batch_size]
            
            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/targets/{target['id']}/websites/bulk-upsert",
                    {"websites": batch}
                )
                
                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))
                
            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"targetId": target["id"], "websites": batch})
    
    progress.finish_phase()
    
    # Create subdomains (only for domain targets)
    domain_targets = [t for t in targets if t["type"] == "domain"]
    subdomain_count = len(domain_targets) * assets_per_target
    
    if domain_targets:
        progress.start_phase("Creating subdomains", subdomain_count, "📝")
        
        for target in domain_targets:
            subdomains = DataGenerator.generate_subdomains(target, assets_per_target)
            
            if not subdomains:
                continue
            
            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/targets/{target['id']}/subdomains/bulk-create",
                    {"names": subdomains}
                )
                
                progress.add_success(len(subdomains))
                progress.update(progress.current_count + len(subdomains))
                
            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"targetId": target["id"], "names": subdomains})
        
        progress.finish_phase()
    
    # Create endpoints
    progress.start_phase(f"Creating endpoints", len(targets) * assets_per_target, "🔗")
    
    for target in targets:
        endpoints = DataGenerator.generate_endpoints(target, assets_per_target)
        
        # Batch create
        for i in range(0, len(endpoints), batch_size):
            batch = endpoints[i:i + batch_size]
            
            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/targets/{target['id']}/endpoints/bulk-upsert",
                    {"endpoints": batch}
                )
                
                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))
                
            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"targetId": target["id"], "endpoints": batch})
    
    progress.finish_phase()
    
    # Create directories
    progress.start_phase(f"Creating directories", len(targets) * assets_per_target, "📁")
    
    for target in targets:
        directories = DataGenerator.generate_directories(target, assets_per_target)
        
        # Batch create
        for i in range(0, len(directories), batch_size):
            batch = directories[i:i + batch_size]
            
            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/targets/{target['id']}/directories/bulk-upsert",
                    {"directories": batch}
                )
                
                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))
                
            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"targetId": target["id"], "directories": batch})
    
    progress.finish_phase()
    
    # Create host ports
    progress.start_phase("Creating host port mappings", len(targets) * assets_per_target, "🔌")
    
    for target in targets:
        host_ports = DataGenerator.generate_host_ports(target, assets_per_target)
        
        # Batch create
        for i in range(0, len(host_ports), batch_size):
            batch = host_ports[i:i + batch_size]
            
            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/targets/{target['id']}/host-ports/bulk-upsert",
                    {"mappings": batch}
                )
                
                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))
                
            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"targetId": target["id"], "mappings": batch})
    
    progress.finish_phase()

    # Create vulnerabilities
    progress.start_phase("Creating vulnerabilities", len(targets) * assets_per_target, "🔓")

    for target in targets:
        vulnerabilities = DataGenerator.generate_vulnerabilities(target, assets_per_target)

        # Batch create
        for i in range(0, len(vulnerabilities), batch_size):
            batch = vulnerabilities[i:i + batch_size]

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/targets/{target['id']}/vulnerabilities/bulk-create",
                    {"vulnerabilities": batch}
                )

                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(
                    str(e),
                    {"targetId": target["id"], "vulnerabilities": batch}
                )

    progress.finish_phase()

    # Create screenshots
    progress.start_phase("Creating screenshots", len(targets) * assets_per_target, "📸")

    for target in targets:
        screenshots = DataGenerator.generate_screenshots(target, assets_per_target)

        # Batch upsert
        for i in range(0, len(screenshots), batch_size):
            batch = screenshots[i:i + batch_size]

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/targets/{target['id']}/screenshots/bulk-upsert",
                    {"screenshots": batch}
                )

                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(
                    str(e),
                    {"targetId": target["id"], "screenshots": batch}
                )

    progress.finish_phase()


def create_scans(client, progress, error_handler, targets):
    """
    Create scans for targets.

    Args:
        client: API client
        progress: Progress tracker
        error_handler: Error handler
        targets: List of target dictionaries

    Returns:
        Dict mapping target_id to scan_id
    """
    from data_generator import DataGenerator

    progress.start_phase("Creating scans", len(targets), "🔍")

    scan_ids = {}

    for target in targets:
        try:
            scan_data = DataGenerator.generate_scan(target)

            result = error_handler.retry_with_backoff(
                client.post,
                "/api/scans/initiate",
                scan_data
            )

            if result.get("scans") and len(result["scans"]) > 0:
                scan_ids[target["id"]] = result["scans"][0]["id"]

            progress.add_success(1)
            progress.update(len(scan_ids))

        except Exception as e:
            progress.add_error(str(e))
            error_handler.log_error(str(e), {"targetId": target["id"]})

    progress.finish_phase()
    return scan_ids


def create_snapshots(client, progress, error_handler, targets, scan_ids, assets_per_target, batch_size):
    """
    Create snapshot data for scans.

    Args:
        client: API client
        progress: Progress tracker
        error_handler: Error handler
        targets: List of target dictionaries
        scan_ids: Dict mapping target_id to scan_id
        assets_per_target: Number of assets per target
        batch_size: Batch size for bulk operations
    """
    from data_generator import DataGenerator

    # Filter targets that have scans
    targets_with_scans = [t for t in targets if t["id"] in scan_ids]

    if not targets_with_scans:
        return

    # Create website snapshots
    progress.start_phase("Creating website snapshots", len(targets_with_scans) * assets_per_target, "🌐")

    for target in targets_with_scans:
        scan_id = scan_ids[target["id"]]
        websites = DataGenerator.generate_websites(target, assets_per_target)

        for i in range(0, len(websites), batch_size):
            batch = websites[i:i + batch_size]

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/scans/{scan_id}/websites/bulk-upsert",
                    {"targetId": target["id"], "websites": batch}
                )

                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"scanId": scan_id, "websites": batch})

    progress.finish_phase()

    # Create subdomain snapshots (only for domain targets)
    domain_targets = [t for t in targets_with_scans if t["type"] == "domain"]

    if domain_targets:
        progress.start_phase("Creating subdomain snapshots", len(domain_targets) * assets_per_target, "📝")

        for target in domain_targets:
            scan_id = scan_ids[target["id"]]
            subdomains = DataGenerator.generate_subdomain_snapshots(target, assets_per_target)

            if not subdomains:
                continue

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/scans/{scan_id}/subdomains/bulk-upsert",
                    {"targetId": target["id"], "subdomains": subdomains}
                )

                progress.add_success(len(subdomains))
                progress.update(progress.current_count + len(subdomains))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"scanId": scan_id, "subdomains": subdomains})

        progress.finish_phase()

    # Create endpoint snapshots
    progress.start_phase("Creating endpoint snapshots", len(targets_with_scans) * assets_per_target, "🔗")

    for target in targets_with_scans:
        scan_id = scan_ids[target["id"]]
        endpoints = DataGenerator.generate_endpoints(target, assets_per_target)

        for i in range(0, len(endpoints), batch_size):
            batch = endpoints[i:i + batch_size]

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/scans/{scan_id}/endpoints/bulk-upsert",
                    {"targetId": target["id"], "endpoints": batch}
                )

                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"scanId": scan_id, "endpoints": batch})

    progress.finish_phase()

    # Create directory snapshots
    progress.start_phase("Creating directory snapshots", len(targets_with_scans) * assets_per_target, "📁")

    for target in targets_with_scans:
        scan_id = scan_ids[target["id"]]
        directories = DataGenerator.generate_directories(target, assets_per_target)

        for i in range(0, len(directories), batch_size):
            batch = directories[i:i + batch_size]

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/scans/{scan_id}/directories/bulk-upsert",
                    {"targetId": target["id"], "directories": batch}
                )

                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"scanId": scan_id, "directories": batch})

    progress.finish_phase()

    # Create host port snapshots
    progress.start_phase("Creating host port snapshots", len(targets_with_scans) * assets_per_target, "🔌")

    for target in targets_with_scans:
        scan_id = scan_ids[target["id"]]
        host_ports = DataGenerator.generate_host_port_snapshots(target, assets_per_target)

        for i in range(0, len(host_ports), batch_size):
            batch = host_ports[i:i + batch_size]

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/scans/{scan_id}/host-ports/bulk-upsert",
                    {"targetId": target["id"], "hostPorts": batch}
                )

                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"scanId": scan_id, "hostPorts": batch})

    progress.finish_phase()

    # Create screenshot snapshots
    progress.start_phase("Creating screenshot snapshots", len(targets_with_scans) * assets_per_target, "📸")

    for target in targets_with_scans:
        scan_id = scan_ids[target["id"]]
        screenshots = DataGenerator.generate_screenshots(target, assets_per_target)

        for i in range(0, len(screenshots), batch_size):
            batch = screenshots[i:i + batch_size]

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/scans/{scan_id}/screenshots/bulk-upsert",
                    {"targetId": target["id"], "screenshots": batch}
                )

                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"scanId": scan_id, "screenshots": batch})

    progress.finish_phase()

    # Create vulnerability snapshots
    progress.start_phase("Creating vulnerability snapshots", len(targets_with_scans) * assets_per_target, "🔓")

    for target in targets_with_scans:
        scan_id = scan_ids[target["id"]]
        vulnerabilities = DataGenerator.generate_vulnerability_snapshots(target, assets_per_target)

        for i in range(0, len(vulnerabilities), batch_size):
            batch = vulnerabilities[i:i + batch_size]

            try:
                error_handler.retry_with_backoff(
                    client.post,
                    f"/api/scans/{scan_id}/vulnerabilities/bulk-create",
                    {"vulnerabilities": batch}
                )

                progress.add_success(len(batch))
                progress.update(progress.current_count + len(batch))

            except Exception as e:
                progress.add_error(str(e))
                error_handler.log_error(str(e), {"scanId": scan_id, "vulnerabilities": batch})

    progress.finish_phase()


if __name__ == "__main__":
    main()
