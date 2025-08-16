#!/usr/bin/env python3
"""
Test script to verify file upload and download functionality
"""
import requests
import hashlib
import os

def create_test_file(filename, size_bytes):
    """Create a test file with specific content"""
    with open(filename, 'wb') as f:
        # Create predictable content for verification
        content = b'A' * (size_bytes // 2) + b'B' * (size_bytes - (size_bytes // 2))
        f.write(content)
    return content

def get_file_hash(content):
    """Get SHA256 hash of file content"""
    return hashlib.sha256(content).hexdigest()

def upload_file(filename):
    """Upload file to FrostByte server"""
    try:
        with open(filename, 'rb') as f:
            response = requests.post(
                f'http://localhost:8080/upload?filename={filename}',
                data=f,
                headers={'Content-Type': 'application/octet-stream'}
            )
        return response.status_code == 200, response.text
    except Exception as e:
        return False, str(e)

def download_file(filename):
    """Download file from FrostByte server"""
    try:
        response = requests.get(f'http://localhost:8080/download/{filename}')
        if response.status_code == 200:
            return True, response.content
        else:
            return False, f"HTTP {response.status_code}: {response.text}"
    except Exception as e:
        return False, str(e)

def test_file_integrity(original_content, downloaded_content):
    """Test if the downloaded file matches the original"""
    original_hash = get_file_hash(original_content)
    downloaded_hash = get_file_hash(downloaded_content)
    
    print(f"Original size: {len(original_content)} bytes")
    print(f"Downloaded size: {len(downloaded_content)} bytes")
    print(f"Original hash: {original_hash}")
    print(f"Downloaded hash: {downloaded_hash}")
    print(f"Integrity check: {'PASS' if original_hash == downloaded_hash else 'FAIL'}")
    
    return original_hash == downloaded_hash

def main():
    # Test files with different sizes
    test_cases = [
        ("small_file.txt", 1024),          # 1KB - smaller than buffer
        ("medium_file.txt", 100 * 1024),   # 100KB - larger than buffer, smaller than chunk
        ("large_file.txt", 15 * 1024 * 1024),  # 15MB - larger than chunk size
    ]
    
    all_passed = True
    
    for filename, size in test_cases:
        print(f"\n{'='*50}")
        print(f"Testing {filename} ({size} bytes)")
        print(f"{'='*50}")
        
        # Create test file
        print(f"Creating test file: {filename}")
        original_content = create_test_file(filename, size)
        
        # Upload file
        print(f"Uploading {filename}...")
        success, message = upload_file(filename)
        if not success:
            print(f"Upload failed: {message}")
            all_passed = False
            continue
        else:
            print(f"Upload successful: {message}")
        
        # Download file
        print(f"Downloading {filename}...")
        success, downloaded_content = download_file(filename)
        if not success:
            print(f"Download failed: {downloaded_content}")
            all_passed = False
            continue
        else:
            print("Download successful")
        
        # Test integrity
        print("Testing file integrity...")
        if test_file_integrity(original_content, downloaded_content):
            print("✓ File integrity test PASSED")
        else:
            print("✗ File integrity test FAILED")
            all_passed = False
        
        # Cleanup
        os.remove(filename)
    
    print(f"\n{'='*60}")
    print(f"Overall test result: {'ALL TESTS PASSED' if all_passed else 'SOME TESTS FAILED'}")
    print(f"{'='*60}")

if __name__ == "__main__":
    main()
