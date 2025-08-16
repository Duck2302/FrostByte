#!/usr/bin/env python3
"""
Test video file upload and download specifically
"""
import requests
import hashlib
import os
import random

def create_fake_video_file(filename, size_mb):
    """Create a fake video file with binary content that mimics video data"""
    size_bytes = size_mb * 1024 * 1024
    
    with open(filename, 'wb') as f:
        # Create video-like binary content with headers
        # MP4 header-like content
        f.write(b'ftypisom')  # MP4 file signature
        f.write(b'\x00\x00\x02\x00')  # version
        f.write(b'isommp41')  # compatible brands
        
        # Fill rest with pseudo-random binary content
        remaining = size_bytes - 16
        written = 0
        while written < remaining:
            chunk_size = min(8192, remaining - written)
            # Create predictable but varied content
            chunk = bytes([(i + written) % 256 for i in range(chunk_size)])
            f.write(chunk)
            written += chunk_size

def upload_and_test_video(filename, size_mb):
    """Upload and test video file integrity"""
    print(f"\n{'='*60}")
    print(f"Testing video file: {filename} ({size_mb}MB)")
    print(f"{'='*60}")
    
    # Create fake video file
    print(f"Creating fake video file: {filename}")
    create_fake_video_file(filename, size_mb)
    
    # Get original file hash
    with open(filename, 'rb') as f:
        original_content = f.read()
    original_hash = hashlib.sha256(original_content).hexdigest()
    print(f"Original file size: {len(original_content)} bytes")
    print(f"Original hash: {original_hash}")
    
    # Upload file
    print(f"Uploading {filename}...")
    try:
        with open(filename, 'rb') as f:
            response = requests.post(
                f'http://localhost:8080/upload?filename={filename}',
                data=f,
                headers={'Content-Type': 'video/mp4'}
            )
        
        if response.status_code == 200:
            print(f"✓ Upload successful: {response.text}")
        else:
            print(f"✗ Upload failed: HTTP {response.status_code} - {response.text}")
            return False
    except Exception as e:
        print(f"✗ Upload error: {e}")
        return False
    
    # Download file
    print(f"Downloading {filename}...")
    try:
        response = requests.get(f'http://localhost:8080/download/{filename}')
        if response.status_code == 200:
            downloaded_content = response.content
            print("✓ Download successful")
        else:
            print(f"✗ Download failed: HTTP {response.status_code} - {response.text}")
            return False
    except Exception as e:
        print(f"✗ Download error: {e}")
        return False
    
    # Test integrity
    downloaded_hash = hashlib.sha256(downloaded_content).hexdigest()
    print(f"Downloaded file size: {len(downloaded_content)} bytes")
    print(f"Downloaded hash: {downloaded_hash}")
    
    # Compare
    size_match = len(original_content) == len(downloaded_content)
    hash_match = original_hash == downloaded_hash
    
    print(f"Size match: {'✓ PASS' if size_match else '✗ FAIL'}")
    print(f"Hash match: {'✓ PASS' if hash_match else '✗ FAIL'}")
    print(f"Overall integrity: {'✓ PASS' if size_match and hash_match else '✗ FAIL'}")
    
    # Cleanup
    os.remove(filename)
    
    return size_match and hash_match

def main():
    """Test various video file sizes"""
    test_cases = [
        ("small_video.mp4", 5),   # 5MB
        ("medium_video.mp4", 25), # 25MB - multi-chunk
        ("large_video.mp4", 50),  # 50MB - many chunks
    ]
    
    all_passed = True
    
    for filename, size_mb in test_cases:
        passed = upload_and_test_video(filename, size_mb)
        if not passed:
            all_passed = False
    
    print(f"\n{'='*60}")
    print(f"Final result: {'ALL VIDEO TESTS PASSED ✓' if all_passed else 'SOME VIDEO TESTS FAILED ✗'}")
    print(f"{'='*60}")

if __name__ == "__main__":
    main()
