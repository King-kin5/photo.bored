document.getElementById('registrationForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const submitBtn = document.getElementById('submit-btn');
    const resultDiv = document.getElementById('registration-result');
    
    // Disable button and show loading state
    submitBtn.disabled = true;
    submitBtn.textContent = 'Signing Up...';
    resultDiv.innerHTML = '';
    
    const formData = {
        username: document.getElementById('username').value,
        email: document.getElementById('email').value,
        password: document.getElementById('password').value
    };

    try {
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(formData),
            credentials: 'include' // Include cookies for authentication
        });

        const data = await response.json();

        if (response.status === 201) {
            // Registration successful
            resultDiv.innerHTML = '<p style="color: green;">Registration successful! Redirecting...</p>';
            
            // Tokens are now set as HTTP-only cookies by the server
            // No need to store in localStorage
            
            // Redirect to feed after a short delay
            setTimeout(() => {
                window.location.href = '/feed';
            }, 1000);
        } else {
            // Registration failed
            resultDiv.innerHTML = `<p style="color: red;">${data.error || "Registration failed. Please try again."}</p>`;
            submitBtn.disabled = false;
            submitBtn.textContent = 'Sign Up';
        }
    } catch (error) {
        console.error('Registration error:', error);
        resultDiv.innerHTML = '<p style="color: red;">An error occurred. Please try again.</p>';
        submitBtn.disabled = false;
        submitBtn.textContent = 'Sign Up';
    }
});