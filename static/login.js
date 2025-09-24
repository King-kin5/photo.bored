document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const submitBtn = document.getElementById('submit-btn');
    const resultDiv = document.getElementById('login-result');
    
    // Disable button and show loading state
    submitBtn.disabled = true;
    submitBtn.textContent = 'Logging In...';
    resultDiv.innerHTML = '';
    
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    
    try {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ email, password })
        });

        const data = await response.json();

        if (response.ok) {
            // Login successful
            resultDiv.innerHTML = '<p style="color: green;">Login successful! Redirecting to feed...</p>';
            
            // Note: Tokens are now set as HTTP-only cookies by the server
            // No need to store in localStorage
            
            // Redirect to feed after successful login
            setTimeout(() => {
                window.location.href = '/feed';
            }, 1000);
        } else {
            // Login failed
            resultDiv.innerHTML = `<p style="color: red;">${data.error || "Login failed. Please check your credentials."}</p>`;
            submitBtn.disabled = false;
            submitBtn.textContent = 'Log In';
        }
    } catch (error) {
        console.error('Login error:', error);
        resultDiv.innerHTML = '<p style="color: red;">An error occurred. Please try again.</p>';
        submitBtn.disabled = false;
        submitBtn.textContent = 'Log In';
    }
});

// Check if user is already logged in by making a request to a protected endpoint
document.addEventListener('DOMContentLoaded', async () => {
    try {
        const response = await fetch('/api/user/profile', {
            method: 'GET',
            credentials: 'include' // Include cookies
        });
        
        if (response.ok) {
            // User is already logged in, redirect to feed
            window.location.href = '/feed';
        }
    } catch (error) {
        // User is not logged in, stay on login page
        console.log('User not logged in');
    }
});