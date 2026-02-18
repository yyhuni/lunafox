"""
Error Handler Module

Handles API errors, retry logic, and error logging.
"""

import time
from typing import Optional, Callable, Any


class ErrorHandler:
    """Handles errors and retry logic for API calls."""
    
    def __init__(self, max_retries: int = 3, retry_delay: float = 1.0):
        """
        Initialize error handler.
        
        Args:
            max_retries: Maximum number of retries for failed requests
            retry_delay: Delay in seconds between retries
        """
        self.max_retries = max_retries
        self.retry_delay = retry_delay
        
    def should_retry(self, status_code: int) -> bool:
        """
        Determine if a request should be retried based on status code.
        
        Args:
            status_code: HTTP status code
            
        Returns:
            True if should retry, False otherwise
        """
        # Retry on 5xx server errors
        if 500 <= status_code < 600:
            return True
        
        # Retry on 429 rate limit
        if status_code == 429:
            return True
        
        # Don't retry on 4xx client errors (except 429)
        return False

    
    def handle_error(self, error: Exception, context: dict) -> bool:
        """
        Handle error and determine if operation should continue.
        
        Args:
            error: The exception that occurred
            context: Context information (endpoint, data, etc.)
            
        Returns:
            True if should continue, False if should stop
        """
        from api_client import APIError
        
        # Handle API errors
        if isinstance(error, APIError):
            if error.status_code and self.should_retry(error.status_code):
                return True  # Continue (will be retried)
            else:
                # Log and skip this record
                self.log_error(str(error), context.get("request_data"), context.get("response_data"))
                return True  # Continue with next record
        
        # Handle network errors (timeout, connection error)
        if isinstance(error, Exception) and "Network error" in str(error):
            return True  # Continue (will be retried)
        
        # Unknown error - log and continue
        self.log_error(str(error), context.get("request_data"))
        return True
    
    def retry_with_backoff(self, func: Callable, *args, **kwargs) -> Any:
        """
        Execute function with retry and exponential backoff.
        
        Args:
            func: Function to execute
            *args: Positional arguments for function
            **kwargs: Keyword arguments for function
            
        Returns:
            Function result
            
        Raises:
            Exception: If all retries fail
        """
        from api_client import APIError
        
        last_error = None
        
        for attempt in range(self.max_retries + 1):
            try:
                return func(*args, **kwargs)
            except APIError as e:
                last_error = e
                
                # Don't retry if not a retryable error
                if e.status_code and not self.should_retry(e.status_code):
                    raise
                
                # Don't retry on last attempt
                if attempt >= self.max_retries:
                    raise
                
                # Calculate delay (longer for rate limit)
                delay = 5.0 if e.status_code == 429 else self.retry_delay
                
                print(f"   ⚠ Retry {attempt + 1}/{self.max_retries} after {delay}s (Status: {e.status_code})")
                time.sleep(delay)
            except Exception as e:
                last_error = e
                
                # Check if it's a network error
                if "Network error" not in str(e):
                    raise
                
                # Don't retry on last attempt
                if attempt >= self.max_retries:
                    raise
                
                print(f"   ⚠ Retry {attempt + 1}/{self.max_retries} after {self.retry_delay}s (Network error)")
                time.sleep(self.retry_delay)
        
        # Should not reach here, but just in case
        if last_error:
            raise last_error

    
    def log_error(self, error: str, request_data: Optional[dict] = None, response_data: Optional[dict] = None):
        """
        Log error details to file.
        
        Args:
            error: Error message
            request_data: Request data (if available)
            response_data: Response data (if available)
        """
        import json
        from datetime import datetime
        
        log_entry = {
            "timestamp": datetime.now().isoformat(),
            "error": error,
        }
        
        if request_data:
            log_entry["request"] = request_data
        
        if response_data:
            log_entry["response"] = response_data
        
        # Append to log file
        try:
            with open("seed_errors.log", "a", encoding="utf-8") as f:
                f.write(json.dumps(log_entry, ensure_ascii=False) + "\n")
        except Exception as e:
            print(f"   ⚠ Failed to write error log: {e}")
