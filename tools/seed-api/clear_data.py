#!/usr/bin/env python3
"""
Database Cleanup Script

Clears all test data from the database via API calls.
"""

import sys
import argparse
from api_client import APIClient, APIError


def parse_args():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Clear all test data from database",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Clear all data using default credentials
  python clear_data.py
  
  # Use custom API URL
  python clear_data.py --api-url http://192.168.1.100:8080
  
  # Use custom credentials
  python clear_data.py --username admin --password mypassword
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
        "--yes",
        action="store_true",
        help="Skip confirmation prompt"
    )
    
    return parser.parse_args()


def confirm_deletion(skip_confirm):
    """Ask user to confirm deletion."""
    if skip_confirm:
        return True
    
    print("⚠️  WARNING: This will delete ALL data from the database!")
    print("   - All organizations")
    print("   - All targets")
    print("   - All assets (websites, subdomains, endpoints, etc.)")
    print()
    response = input("Are you sure you want to continue? (yes/no): ")
    return response.lower() in ['yes', 'y']


def clear_all_data(client):
    """
    Clear all data from database.
    
    Args:
        client: API client
    """
    print("🗑️  Clearing all data...")
    print()
    
    # Delete in correct order (child tables first)
    delete_operations = [
        ("vulnerabilities", "/api/vulnerabilities/bulk-delete"),
        ("host ports", "/api/host-ports/bulk-delete"),
        ("directories", "/api/directories/bulk-delete"),
        ("endpoints", "/api/endpoints/bulk-delete"),
        ("subdomains", "/api/subdomains/bulk-delete"),
        ("websites", "/api/websites/bulk-delete"),
        ("targets", "/api/targets/bulk-delete"),
        ("organizations", "/api/organizations/bulk-delete"),
    ]
    
    total_deleted = 0
    
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
                total_deleted += len(ids)
            else:
                print(f"   ✓ No {name} to delete")
                
        except Exception as e:
            print(f"   ⚠ Failed to delete {name}: {e}")
    
    print()
    print(f"✅ Cleanup completed! Deleted {total_deleted} records total.")


def main():
    """Main entry point."""
    args = parse_args()
    
    # Confirm deletion
    if not confirm_deletion(args.yes):
        print("❌ Cancelled by user")
        sys.exit(0)
    
    # Initialize API client
    client = APIClient(args.api_url, args.username, args.password)
    
    try:
        # Login
        print("🔐 Logging in...")
        client.login()
        print("   ✓ Authenticated")
        print()
        
        # Clear all data
        clear_all_data(client)
        
    except APIError as e:
        print(f"\n❌ API Error: {e}")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n\n⚠ Interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
