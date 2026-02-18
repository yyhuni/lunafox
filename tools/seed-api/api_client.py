"""
API Client Module

Handles HTTP requests, authentication, and token management.
"""

import requests
from typing import Optional, Dict, Any


class APIError(Exception):
    """Custom exception for API errors."""
    
    def __init__(self, message: str, status_code: Optional[int] = None, response_data: Optional[Dict] = None):
        super().__init__(message)
        self.status_code = status_code
        self.response_data = response_data


class APIClient:
    """API client for interacting with the Go backend."""
    
    def __init__(self, base_url: str, username: str, password: str):
        """
        Initialize API client.
        
        Args:
            base_url: Base URL of the API (e.g., http://localhost:8080)
            username: Username for authentication
            password: Password for authentication
        """
        self.base_url = base_url.rstrip('/')
        self.username = username
        self.password = password
        self.session = requests.Session()
        self.access_token: Optional[str] = None
        self.refresh_token_value: Optional[str] = None
        
    def login(self) -> str:
        """
        Login and get JWT token.
        
        Returns:
            Access token
            
        Raises:
            requests.HTTPError: If login fails
        """
        url = f"{self.base_url}/api/auth/login"
        data = {
            "username": self.username,
            "password": self.password
        }
        
        response = self.session.post(url, json=data, timeout=30)
        response.raise_for_status()
        
        result = response.json()
        self.access_token = result["accessToken"]
        self.refresh_token_value = result["refreshToken"]
        
        return self.access_token
    
    def refresh_token(self) -> str:
        """
        Refresh expired token.
        
        Returns:
            New access token
            
        Raises:
            requests.HTTPError: If refresh fails
        """
        url = f"{self.base_url}/api/auth/refresh"
        data = {
            "refreshToken": self.refresh_token_value
        }
        
        response = self.session.post(url, json=data, timeout=30)
        response.raise_for_status()
        
        result = response.json()
        self.access_token = result["accessToken"]
        
        return self.access_token
    
    def _get_headers(self) -> Dict[str, str]:
        """
        Get request headers with authorization.
        
        Returns:
            Headers dictionary
        """
        headers = {
            "Content-Type": "application/json"
        }
        
        if self.access_token:
            headers["Authorization"] = f"Bearer {self.access_token}"
            
        return headers

    
    def _handle_error(self, error: requests.HTTPError) -> None:
        """
        Parse and raise API error with detailed information.
        
        Args:
            error: HTTP error from requests
            
        Raises:
            APIError: With parsed error message
        """
        try:
            error_data = error.response.json()
            if "error" in error_data:
                error_info = error_data["error"]
                message = error_info.get("message", str(error))
                code = error_info.get("code", "UNKNOWN")
                raise APIError(
                    f"API Error [{code}]: {message}",
                    status_code=error.response.status_code,
                    response_data=error_data
                )
        except (ValueError, KeyError):
            # If response is not JSON or doesn't have expected structure
            pass
        
        # Fallback to original error
        raise APIError(
            str(error),
            status_code=error.response.status_code if error.response else None
        )
    
    def post(self, endpoint: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Send POST request with automatic token refresh on 401.
        
        Args:
            endpoint: API endpoint (e.g., /api/targets)
            data: Request data (will be JSON encoded)
            
        Returns:
            Response data (JSON decoded)
            
        Raises:
            requests.HTTPError: If request fails
        """
        url = f"{self.base_url}{endpoint}"
        headers = self._get_headers()
        
        try:
            response = self.session.post(url, json=data, headers=headers, timeout=30)
            response.raise_for_status()
        except requests.HTTPError as e:
            # Auto refresh token on 401
            if e.response.status_code == 401 and self.refresh_token_value:
                self.refresh_token()
                headers = self._get_headers()
                try:
                    response = self.session.post(url, json=data, headers=headers, timeout=30)
                    response.raise_for_status()
                except requests.HTTPError as retry_error:
                    self._handle_error(retry_error)
            else:
                self._handle_error(e)
        except (requests.Timeout, requests.ConnectionError) as e:
            raise APIError(f"Network error: {str(e)}")
        
        # Handle 204 No Content
        if response.status_code == 204:
            return {}
            
        return response.json()
    
    def get(self, endpoint: str, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Send GET request with automatic token refresh on 401.
        
        Args:
            endpoint: API endpoint (e.g., /api/targets)
            params: Query parameters
            
        Returns:
            Response data (JSON decoded)
            
        Raises:
            requests.HTTPError: If request fails
        """
        url = f"{self.base_url}{endpoint}"
        headers = self._get_headers()
        
        try:
            response = self.session.get(url, params=params, headers=headers, timeout=30)
            response.raise_for_status()
        except requests.HTTPError as e:
            # Auto refresh token on 401
            if e.response.status_code == 401 and self.refresh_token_value:
                self.refresh_token()
                headers = self._get_headers()
                try:
                    response = self.session.get(url, params=params, headers=headers, timeout=30)
                    response.raise_for_status()
                except requests.HTTPError as retry_error:
                    self._handle_error(retry_error)
            else:
                self._handle_error(e)
        except (requests.Timeout, requests.ConnectionError) as e:
            raise APIError(f"Network error: {str(e)}")
        
        return response.json()
    
    def delete(self, endpoint: str) -> None:
        """
        Send DELETE request with automatic token refresh on 401.
        
        Args:
            endpoint: API endpoint (e.g., /api/targets/1)
            
        Raises:
            requests.HTTPError: If request fails
        """
        url = f"{self.base_url}{endpoint}"
        headers = self._get_headers()
        
        try:
            response = self.session.delete(url, headers=headers, timeout=30)
            response.raise_for_status()
        except requests.HTTPError as e:
            # Auto refresh token on 401
            if e.response.status_code == 401 and self.refresh_token_value:
                self.refresh_token()
                headers = self._get_headers()
                try:
                    response = self.session.delete(url, headers=headers, timeout=30)
                    response.raise_for_status()
                except requests.HTTPError as retry_error:
                    self._handle_error(retry_error)
            else:
                self._handle_error(e)
        except (requests.Timeout, requests.ConnectionError) as e:
            raise APIError(f"Network error: {str(e)}")
