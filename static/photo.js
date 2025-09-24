// Photo Upload Page JavaScript - CSP Compliant
let selectedFiles = [];

document.addEventListener('DOMContentLoaded', async () => {
    // Check if user is authenticated by calling the profile endpoint
    await checkAuthentication();
});

// FIXED: Use proper authentication check with cookies
async function checkAuthentication() {
    try {
        const response = await fetch('/api/user/profile', {
            method: 'GET',
            credentials: 'include' // Include cookies
        });
        
        if (response.ok) {
            // User is authenticated, initialize the upload page
            console.log('User authenticated, initializing upload page');
            initializeUploadPage();
        } else {
            // User is not authenticated, redirect to login
            console.log('User not authenticated, redirecting to login');
            window.location.href = '/login_page';
            return;
        }
    } catch (error) {
        console.error('Authentication check failed:', error);
        window.location.href = '/login_page';
    }
}

function initializeUploadPage() {
    const uploadArea = document.getElementById('upload-area');
    const fileUpload = document.getElementById('file-upload');
    const previewContainer = document.getElementById('preview-container');
    const uploadButton = document.getElementById('upload-button');

    // Click to select files
    uploadArea.addEventListener('click', () => fileUpload.click());

    // Drag and drop functionality
    uploadArea.addEventListener('dragover', (e) => {
        e.preventDefault();
        uploadArea.style.borderColor = 'var(--primary-color)';
    });

    uploadArea.addEventListener('dragleave', () => {
        uploadArea.style.borderColor = 'var(--accent-color)';
    });

    uploadArea.addEventListener('drop', (e) => {
        e.preventDefault();
        uploadArea.style.borderColor = 'var(--accent-color)';
        const files = e.dataTransfer.files;
        handleFiles(files);
    });

    // File input change
    fileUpload.addEventListener('change', () => {
        const files = fileUpload.files;
        handleFiles(files);
    });

    // Upload button click
    uploadButton.addEventListener('click', uploadPhotos);

    // Logout button
    document.getElementById('logout-btn').addEventListener('click', logout);

    // Preview container for remove buttons
    previewContainer.addEventListener('click', (e) => {
        if (e.target.classList.contains('remove-preview')) {
            const filename = e.target.dataset.filename;
            selectedFiles = selectedFiles.filter(file => file.name !== filename);
            e.target.parentElement.remove();
        }
    });
}

function handleFiles(files) {
    for (const file of files) {
        if (file.type.startsWith('image/')) {
            // Check file size (10MB limit)
            if (file.size > 10 * 1024 * 1024) {
                alert(`File "${file.name}" is too large. Maximum size is 10MB.`);
                continue;
            }
            
            // Check for duplicates
            const isDuplicate = selectedFiles.some(existingFile => 
                existingFile.name === file.name && existingFile.size === file.size
            );
            
            if (!isDuplicate) {
                selectedFiles.push(file);
                createPreviewItem(file);
            } else {
                console.log(`File "${file.name}" already selected`);
            }
        } else {
            alert(`File "${file.name}" is not a valid image file.`);
        }
    }
}

function createPreviewItem(file) {
    const reader = new FileReader();
    const previewContainer = document.getElementById('preview-container');
    
    reader.onload = (e) => {
        const previewItem = document.createElement('div');
        previewItem.className = 'preview-item';
        
        const img = document.createElement('img');
        img.src = e.target.result;
        img.alt = file.name;
        img.style.maxWidth = '200px';
        img.style.maxHeight = '200px';
        img.style.objectFit = 'cover';
        
        const removeBtn = document.createElement('button');
        removeBtn.className = 'remove-preview';
        removeBtn.dataset.filename = file.name;
        removeBtn.textContent = 'X';
        removeBtn.title = 'Remove image';
        removeBtn.style.position = 'absolute';
        removeBtn.style.top = '5px';
        removeBtn.style.right = '5px';
        removeBtn.style.background = 'rgba(255, 0, 0, 0.8)';
        removeBtn.style.color = 'white';
        removeBtn.style.border = 'none';
        removeBtn.style.borderRadius = '50%';
        removeBtn.style.width = '25px';
        removeBtn.style.height = '25px';
        removeBtn.style.cursor = 'pointer';
        
        previewItem.style.position = 'relative';
        previewItem.style.display = 'inline-block';
        previewItem.style.margin = '10px';
        
        previewItem.appendChild(img);
        previewItem.appendChild(removeBtn);
        previewContainer.appendChild(previewItem);
    };
    
    reader.onerror = () => {
        console.error('Error reading file:', file.name);
        alert(`Error reading file: ${file.name}`);
    };
    
    reader.readAsDataURL(file);
}

async function uploadPhotos() {
    if (selectedFiles.length === 0) {
        alert('Please select at least one image to upload');
        return;
    }

    const uploadButton = document.getElementById('upload-button');
    const originalText = uploadButton.textContent;
    
    // Update button state
    uploadButton.disabled = true;
    uploadButton.textContent = 'Uploading...';
    uploadButton.style.opacity = '0.7';

    const formData = new FormData();
    selectedFiles.forEach(file => {
        formData.append('images[]', file);
    });

    // Add any additional form data like captions if available
    const captionInput = document.getElementById('caption-input');
    if (captionInput && captionInput.value.trim()) {
        formData.append('caption', captionInput.value.trim());
    }

    try {
        console.log('Uploading files:', selectedFiles.length);
        
        const response = await fetch('/api/upload', {
            method: 'POST',
            credentials: 'include', // Use cookies instead of Authorization header
            body: formData
        });

        console.log('Upload response status:', response.status);

        if (response.ok) {
            console.log('Upload successful, redirecting to feed');
            // Success - show success message and redirect to feed
            alert('Photos uploaded successfully!');
            window.location.href = '/feed';
        } else if (response.status === 401) {
            console.log('Authentication failed, redirecting to login');
            alert('Session expired. Redirecting to login...');
            window.location.href = '/login_page';
        } else {
            const errorData = await response.json().catch(() => ({}));
            console.error('Upload failed:', errorData);
            alert('Upload failed: ' + (errorData.error || errorData.message || 'Unknown error'));
        }
    } catch (error) {
        console.error('Upload error:', error);
        alert('An error occurred during upload. Please check your connection and try again.');
    } finally {
        // Reset button state
        uploadButton.disabled = false;
        uploadButton.textContent = originalText;
        uploadButton.style.opacity = '1';
    }
}

// FIXED: Logout function using cookies
async function logout() {
    try {
        console.log('Initiating logout');
        // Call the logout endpoint to clear server-side session
        const response = await fetch('/api/logout', {
            method: 'POST',
            credentials: 'include'
        });
        
        if (response.ok) {
            console.log('Logout successful');
        } else {
            console.log('Logout response not OK, but continuing with cleanup');
        }
    } catch (error) {
        console.error('Logout error:', error);
    }
    
    // Clear any localStorage tokens (legacy cleanup)
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    
    console.log('Redirecting to login page');
    window.location.href = '/login_page';
}