"""
Progress Tracker Module

Displays progress and statistics during data generation.
"""

import time
from typing import List


class ProgressTracker:
    """Tracks and displays progress during data generation."""
    
    def __init__(self):
        """Initialize progress tracker."""
        self.current_phase = ""
        self.current_count = 0
        self.total_count = 0
        self.success_count = 0
        self.error_count = 0
        self.errors: List[str] = []
        self.start_time = time.time()
        self.phase_start_time = 0.0
        
    def start_phase(self, phase_name: str, total: int, emoji: str = ""):
        """
        Start a new phase.
        
        Args:
            phase_name: Name of the phase (e.g., "Creating organizations")
            total: Total number of items to process
            emoji: Emoji icon for the phase
        """
        self.current_phase = phase_name
        self.current_count = 0
        self.total_count = total
        self.success_count = 0
        self.error_count = 0
        self.phase_start_time = time.time()
        
        prefix = f"{emoji} " if emoji else ""
        print(f"{prefix}{phase_name}...", end=" ", flush=True)
        
    def update(self, count: int):
        """
        Update progress.
        
        Args:
            count: Current count
        """
        self.current_count = count

    
    def add_success(self, count: int):
        """
        Record successful operations.
        
        Args:
            count: Number of successful operations
        """
        self.success_count += count
        
    def add_error(self, error: str):
        """
        Record an error.
        
        Args:
            error: Error message
        """
        self.error_count += 1
        self.errors.append(error)
        
    def finish_phase(self):
        """Complete current phase and display summary."""
        elapsed = time.time() - self.phase_start_time
        print(f"[{self.current_count}/{self.total_count}] ✓ {self.success_count} created", end="")
        
        if self.error_count > 0:
            print(f" ({self.error_count} errors)", end="")
        
        print()  # New line

    
    def print_summary(self):
        """Print final summary."""
        total_time = time.time() - self.start_time
        
        print()
        print("✅ Test data generation completed!")
        print(f"   Total time: {total_time:.1f}s")
        print(f"   Success: {self.success_count:,} records")
        
        if self.error_count > 0:
            print(f"   Errors: {self.error_count} records")
