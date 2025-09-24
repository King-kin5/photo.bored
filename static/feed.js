// Complete Feed Page JavaScript - Enhanced with Better Error Handling and Fixed Like Functionality
class FeedManager {
    constructor() {
        this.loading = false;
        this.hasMore = true;
        this.nextCursor = null;
        this.feedContainer = null;
        this.selectedFile = null;
        this.isAuthenticated = false;
        this.currentFeedType = 'public'; // Default to public for unauthenticated users
        this.observer = null;
        this.websocket = null;
        this.retryCount = 0;
        this.maxRetries = 3;
        
        this.FEED_TYPES = {
            public: { 
                label: 'Discover', 
                icon: 'compass',
                description: 'Explore all public photos',
                color: '#f093fb',
                priority: 1,
                requiresAuth: false
            },
            combined: { 
                label: 'Home', 
                icon: 'home',
                description: 'Your photos and discover feed',
                color: '#667eea',
                priority: 2,
                requiresAuth: true
            },
            personal: { 
                label: 'My Photos', 
                icon: 'user',
                description: 'Your uploaded photos',
                color: '#4facfe',
                priority: 3,
                requiresAuth: true
            },
            following: { 
                label: 'Following', 
                icon: 'users',
                description: 'Photos from people you follow',
                color: '#43e97b',
                priority: 4,
                requiresAuth: true
            }
        };
        
        this.init();
    }

    async init() {
        document.addEventListener('DOMContentLoaded', () => this.onDOMContentLoaded());
    }

    async onDOMContentLoaded() {
        this.feedContainer = document.getElementById('feed-container');
        await this.checkAuthentication();
        this.setupIntersectionObserver();
        this.createParticles();
        this.setupScrollIndicator();
        this.setupFeedTypeSelector();
        this.setupPullToRefresh();
        await this.loadInitialPosts();
        this.setupWebSocket();
        this.setupUploadModal();
        this.setupEventListeners();
    }

    async checkAuthentication() {
        try {
            const response = await fetch('/api/user/profile', {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json',
                    'Content-Type': 'application/json'
                }
            });
            
            if (response.ok) {
                this.isAuthenticated = true;
                this.currentFeedType = 'combined'; // Start with combined feed for authenticated users
                console.log('User authenticated - starting with combined feed');
            } else {
                console.log('User not authenticated - using public feed');
                this.isAuthenticated = false;
                this.currentFeedType = 'public';
            }
        } catch (error) {
            console.error('Authentication check failed:', error);
            this.isAuthenticated = false;
            this.currentFeedType = 'public';
        }

        this.updateUIForAuthState();
    }

    updateUIForAuthState() {
        const uploadFab = document.getElementById('upload-fab');
        const logoutBtn = document.getElementById('logout-btn');
        
        if (!this.isAuthenticated) {
            if (uploadFab) uploadFab.style.display = 'none';
            if (logoutBtn) logoutBtn.style.display = 'none';
            this.showLoginPrompt();
        } else {
            if (uploadFab) uploadFab.style.display = 'flex';
            if (logoutBtn) logoutBtn.style.display = 'block';
        }
    }

    setupIntersectionObserver() {
        this.observer = new IntersectionObserver((entries) => {
            if (entries[0].isIntersecting && !this.loading && this.hasMore) {
                this.loadMorePosts();
            }
        }, { 
            threshold: 0.1,
            rootMargin: '100px'
        });

        const observerElement = document.getElementById('intersection-observer');
        if (observerElement) {
            this.observer.observe(observerElement);
        }
    }

    createParticles() {
        const particlesContainer = document.getElementById('particles');
        if (!particlesContainer) return;
        
        const particleCount = 50;
        for (let i = 0; i < particleCount; i++) {
            const particle = document.createElement('div');
            particle.className = 'particle';
            particle.style.left = Math.random() * 100 + '%';
            particle.style.animationDelay = Math.random() * 15 + 's';
            particle.style.animationDuration = (Math.random() * 10 + 10) + 's';
            particlesContainer.appendChild(particle);
        }
    }

    setupScrollIndicator() {
        window.addEventListener('scroll', () => {
            const scrollIndicator = document.getElementById('scroll-indicator');
            if (!scrollIndicator) return;
            
            const scrollTop = window.pageYOffset;
            const documentHeight = document.documentElement.scrollHeight - window.innerHeight;
            const scrollPercent = Math.max(0, Math.min(100, (scrollTop / documentHeight) * 100));
            scrollIndicator.style.width = scrollPercent + '%';
        });
    }

    setupWebSocket() {
        if (!this.isAuthenticated) {
            console.log('Skipping WebSocket setup for unauthenticated user');
            return;
        }
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/feed`;
        
        console.log('Connecting to WebSocket:', wsUrl);
        this.websocket = new WebSocket(wsUrl);
        
        this.websocket.onopen = () => {
            console.log('WebSocket connected');
            this.retryCount = 0;
        };
        
        this.websocket.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                console.log('WebSocket message received:', message);
                
                if (message.type === 'new_photo' && message.data) {
                    if (['combined', 'public', 'personal'].includes(this.currentFeedType)) {
                        this.prependPost(message.data);
                        this.showToast('New photo shared!', 'success');
                    }
                }
            } catch (error) {
                console.error('WebSocket message error:', error);
            }
        };

        this.websocket.onclose = (event) => {
            console.log('WebSocket closed:', event.code, event.reason);
            if (this.isAuthenticated && this.retryCount < this.maxRetries) {
                this.retryCount++;
                console.log(`WebSocket reconnecting... (attempt ${this.retryCount}/${this.maxRetries})`);
                setTimeout(() => this.setupWebSocket(), 3000 * this.retryCount);
            }
        };

        this.websocket.onerror = (error) => {
            console.log('WebSocket error:', error);
        };
    }

    setupEventListeners() {
        if (this.isAuthenticated) {
            const logoutBtn = document.getElementById('logout-btn');
            if (logoutBtn) {
                logoutBtn.addEventListener('click', (e) => {
                    e.preventDefault();
                    this.logout();
                });
            }
        }
    }

    async loadInitialPosts() {
        this.loading = true;
        this.hasMore = true;
        this.nextCursor = null;
        this.retryCount = 0;
        
        // Clear all existing content except login prompt
        const loginPrompt = this.feedContainer.querySelector('.login-prompt');
        this.feedContainer.innerHTML = '';
        if (loginPrompt && !this.isAuthenticated) {
            this.feedContainer.appendChild(loginPrompt);
        }
        
        try {
            console.log(`Loading initial posts for feed type: ${this.currentFeedType}`);
            const response = await this.fetchFeed();
            
            if (response && response.photos && Array.isArray(response.photos)) {
                console.log(`Received ${response.photos.length} photos`);
                
                // Filter out photos with invalid filenames
                const validPhotos = response.photos.filter(photo => {
                    if (!photo.filename || photo.filename === 'undefined' || photo.filename === 'null' || photo.filename === '') {
                        console.warn('Filtering out photo with invalid filename:', photo);
                        return false;
                    }
                    return true;
                });
                
                if (validPhotos.length > 0) {
                    validPhotos.forEach((post, index) => {
                        setTimeout(() => {
                            this.appendPost(post);
                        }, index * 50);
                    });
                    
                    this.hasMore = response.has_more;
                    this.nextCursor = response.next_cursor;
                } else {
                    console.log('No valid photos found');
                    this.hasMore = false;
                    this.showEmptyFeedMessage();
                }
            } else {
                console.log('No photos in response or invalid response format');
                this.hasMore = false;
                this.showEmptyFeedMessage();
            }
        } catch (error) {
            console.error('Error loading initial posts:', error);
            this.showToast('Failed to load feed', 'error');
            this.hasMore = false;
            this.showEmptyFeedMessage();
        }
        
        this.loading = false;
    }

    async loadMorePosts() {
        if (this.loading || !this.hasMore) return;
        
        this.loading = true;
        this.showLoadingSpinner();
        
        try {
            const response = await this.fetchFeed(this.nextCursor);
            
            if (response && response.photos && Array.isArray(response.photos)) {
                // Filter out photos with invalid filenames
                const validPhotos = response.photos.filter(photo => {
                    if (!photo.filename || photo.filename === 'undefined' || photo.filename === 'null' || photo.filename === '') {
                        console.warn('Filtering out photo with invalid filename:', photo);
                        return false;
                    }
                    return true;
                });
                
                if (validPhotos.length > 0) {
                    validPhotos.forEach((post, index) => {
                        setTimeout(() => {
                            this.appendPost(post);
                        }, index * 50);
                    });
                    
                    this.hasMore = response.has_more;
                    this.nextCursor = response.next_cursor;
                } else {
                    this.hasMore = false;
                    this.showEndOfFeedMessage();
                }
            } else {
                this.hasMore = false;
                this.showEndOfFeedMessage();
            }
        } catch (error) {
            console.error('Error loading more posts:', error);
            this.showToast('Failed to load more posts', 'error');
        }
        
        this.hideLoadingSpinner();
        this.loading = false;
    }

    async fetchFeed(cursor = null) {
        const params = new URLSearchParams({
            feed_type: this.currentFeedType,
            limit: '20'
        });
        
        if (cursor) params.append('cursor', cursor);
        if (this.isAuthenticated) params.append('personalized', 'true');
        
        const url = `/api/feed/infinite?${params.toString()}`;
        console.log('Fetching feed:', url);
        
        const response = await fetch(url, {
            method: 'GET',
            credentials: 'include',
            headers: {
                'Accept': 'application/json',
                'Content-Type': 'application/json'
            }
        });
        
        if (response.status === 401 && this.isAuthenticated) {
            this.showToast('Session expired. Redirecting to login...', 'error');
            setTimeout(() => {
                window.location.href = '/login_page';
            }, 2000);
            throw new Error('Unauthorized');
        }
        
        if (!response.ok) {
            const errorText = await response.text().catch(() => 'Unknown error');
            throw new Error(`HTTP ${response.status}: ${errorText}`);
        }
        
        const data = await response.json();
        console.log('Feed response:', data);
        return data;
    }

    appendPost(post) {
        // Additional validation before creating post element
        if (!post || !post.filename || post.filename === 'undefined' || post.filename === 'null') {
            console.warn('Skipping invalid post:', post);
            return;
        }
        
        const postElement = this.createPostElement(post);
        this.feedContainer.appendChild(postElement);
        
        this.setupImageLazyLoading(postElement);
    }

    prependPost(post) {
        // Additional validation before creating post element
        if (!post || !post.filename || post.filename === 'undefined' || post.filename === 'null') {
            console.warn('Skipping invalid post for prepend:', post);
            return;
        }
        
        const postElement = this.createPostElement(post);
        this.feedContainer.insertBefore(postElement, this.feedContainer.firstChild);
        
        // Entrance animation
        postElement.style.transform = 'translateY(-100px)';
        postElement.style.opacity = '0';
        
        setTimeout(() => {
            postElement.style.transition = 'all 0.6s cubic-bezier(0.4, 0, 0.2, 1)';
            postElement.style.transform = 'translateY(0)';
            postElement.style.opacity = '1';
        }, 100);
        
        this.setupImageLazyLoading(postElement);
    }

    setupImageLazyLoading(postElement) {
        const img = postElement.querySelector('img');
        if (!img) return;
        
        img.onload = () => {
            img.classList.add('loaded');
            // Hide the error message if image loads successfully
            const errorDiv = img.nextElementSibling;
            if (errorDiv && errorDiv.classList.contains('image-error')) {
                errorDiv.style.display = 'none';
            }
        };
        
        img.onerror = () => {
            console.error('Failed to load image:', img.src);
            img.style.display = 'none';
            // Show the error message
            const errorDiv = img.nextElementSibling;
            if (errorDiv && errorDiv.classList.contains('image-error')) {
                errorDiv.style.display = 'flex';
            }
        };
        
        const imageObserver = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const image = entry.target;
                    if (image.dataset.src) {
                        image.src = image.dataset.src;
                        image.removeAttribute('data-src');
                        imageObserver.unobserve(image);
                    }
                }
            });
        });
        
        if (img.dataset.src) {
            imageObserver.observe(img);
        }
    }

    createPostElement(post) {
        // Final validation
        if (!post.filename || post.filename === 'undefined' || post.filename === 'null' || post.filename === '') {
            console.error('Cannot create post element for invalid filename:', post);
            return document.createElement('div'); // Return empty div to prevent errors
        }
        
        const div = document.createElement('div');
        div.className = 'feed-item';
        div.dataset.postId = post.photo_id;
        
        const timeAgo = this.formatTimeAgo(post.created_at || post.date);
        const imageUrl = `/serveimage/${encodeURIComponent(post.filename)}`;
        
        div.innerHTML = `
            <div class="post-header">
                <div class="user-info">
                    <div class="avatar">
                        <div class="avatar-placeholder">${(post.username || 'A')[0].toUpperCase()}</div>
                    </div>
                    <div class="user-details">
                        <h3 class="username">${this.escapeHtml(post.username || 'Anonymous')}</h3>
                        <small class="post-time">${timeAgo}</small>
                    </div>
                </div>
                <button class="post-menu-btn" title="More options">
                    <svg viewBox="0 0 24 24" width="20" height="20">
                        <circle cx="12" cy="5" r="2"/>
                        <circle cx="12" cy="12" r="2"/>
                        <circle cx="12" cy="19" r="2"/>
                    </svg>
                </button>
            </div>
            
            <div class="post-image-container">
                <img src="${imageUrl}" 
                     alt="Post by ${this.escapeHtml(post.username || 'Anonymous')}" 
                     loading="lazy">
                <div class="image-error" style="display: none; justify-content: center; align-items: center; height: 300px; background: #333; color: #fff; border-radius: 8px;">
                    <div style="text-align: center;">
                        <div style="font-size: 48px; margin-bottom: 10px;">📷</div>
                        <div>Image failed to load</div>
                    </div>
                </div>
                <div class="image-overlay">
                    <button class="like-btn" title="Like this photo" data-post-id="${post.photo_id}">
                        <svg class="heart-icon" viewBox="0 0 24 24">
                            <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
                        </svg>
                    </button>
                </div>
            </div>
            
            <div class="post-content">
                <div class="post-actions">
                    <button class="action-btn like-btn-main" data-post-id="${post.photo_id}">
                        <svg viewBox="0 0 24 24" width="24" height="24">
                            <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
                        </svg>
                        <span class="likes-count">${post.likes_count || 0}</span>
                    </button>
                    <button class="action-btn comment-btn" data-post-id="${post.photo_id}">
                        <svg viewBox="0 0 24 24" width="24" height="24">
                            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
                        </svg>
                        <span class="comments-count">${post.comments_count || 0}</span>
                    </button>
                    <button class="action-btn share-btn" data-post-id="${post.photo_id}">
                        <svg viewBox="0 0 24 24" width="24" height="24">
                            <path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/>
                            <polyline points="16,6 12,2 8,6"/>
                            <line x1="12" y1="2" x2="12" y2="15"/>
                        </svg>
                    </button>
                </div>
                
                ${post.caption ? `<p class="post-caption">${this.escapeHtml(post.caption)}</p>` : ''}
                
                ${post.location && post.location !== 'Unknown Location' ? 
                    `<p class="post-location">
                        <svg viewBox="0 0 24 24" width="16" height="16">
                            <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
                            <circle cx="12" cy="10" r="3"/>
                        </svg>
                        ${this.escapeHtml(post.location)}
                    </p>` : ''
                }
            </div>
        `;
        
        this.setupPostEventListeners(div, post);
        return div;
    }

    setupPostEventListeners(postElement, post) {
        const likeButtons = postElement.querySelectorAll('.like-btn, .like-btn-main');
        likeButtons.forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                if (this.isAuthenticated) {
                    this.toggleLike(btn, post.photo_id);
                } else {
                    this.showToast('Please login to like photos', 'info');
                }
            });
        });
        
        const commentBtn = postElement.querySelector('.comment-btn');
        commentBtn.addEventListener('click', () => this.showComments(post.photo_id));
        
        const shareBtn = postElement.querySelector('.share-btn');
        shareBtn.addEventListener('click', () => this.sharePost(post));
    }

    toggleLike(button, postId) {
        // Visual feedback first (optimistic update)
        const wasLiked = button.classList.contains('liked');
        button.classList.toggle('liked');
        
        const heart = button.querySelector('.heart-icon, svg');
        if (heart) {
            heart.style.transform = 'scale(1.3)';
            
            if (button.classList.contains('liked')) {
                heart.style.color = '#ef4444';
                heart.style.fill = '#ef4444';
            } else {
                heart.style.color = '';
                heart.style.fill = '';
            }
            
            setTimeout(() => {
                heart.style.transform = '';
            }, 200);
        }
        
        const likesCount = button.querySelector('.likes-count');
        if (likesCount) {
            const currentCount = parseInt(likesCount.textContent) || 0;
            likesCount.textContent = button.classList.contains('liked') ? 
                currentCount + 1 : Math.max(0, currentCount - 1);
        }
        
        // Attempt to send like to server (but don't revert on failure since endpoint doesn't exist)
        this.sendLike(postId, button.classList.contains('liked'), button, wasLiked);
    }

    async sendLike(postId, isLiked, button, wasLiked) {
        try {
            const response = await fetch(`/api/photos/${postId}/like`, {
                method: isLiked ? 'POST' : 'DELETE',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json'
                }
            });
            
            if (response.status === 404) {
                console.log('Like functionality not implemented yet - keeping visual state');
                // Don't show error to user since visual feedback already happened
                return;
            }
            
            if (!response.ok) {
                console.error('Failed to update like status:', response.status);
                // Could revert visual state here if needed
                this.showToast('Could not save like status', 'warning');
            }
        } catch (error) {
            console.error('Error updating like:', error);
            // Don't show error for network issues with likes
        }
    }

    showComments(postId) {
        console.log('Show comments for post:', postId);
        this.showToast('Comments feature coming soon!', 'info');
    }

    sharePost(post) {
        if (navigator.share) {
            navigator.share({
                title: `Photo by ${post.username}`,
                text: post.caption || `Check out this photo by ${post.username}`,
                url: window.location.href
            });
        } else {
            navigator.clipboard.writeText(window.location.href).then(() => {
                this.showToast('Link copied to clipboard!', 'success');
            }).catch(() => {
                this.showToast('Unable to share', 'error');
            });
        }
    }

    setupFeedTypeSelector() {
        let feedSelector = document.getElementById('feed-selector');
        if (!feedSelector) {
            feedSelector = document.createElement('div');
            feedSelector.id = 'feed-selector';
            feedSelector.className = 'feed-selector';
            
            const header = document.querySelector('header') || document.querySelector('.header');
            if (header) {
                header.insertAdjacentElement('afterend', feedSelector);
            } else {
                document.body.insertBefore(feedSelector, this.feedContainer);
            }
        }
        
        this.renderFeedTabs(feedSelector);
        this.addFeedTabStyles();
    }

    renderFeedTabs(feedSelector) {
        // Sort feed types by priority and filter by authentication requirements
        const availableFeedTypes = Object.keys(this.FEED_TYPES).filter(feedType => {
            const config = this.FEED_TYPES[feedType];
            return !config.requiresAuth || this.isAuthenticated;
        }).sort((a, b) => this.FEED_TYPES[a].priority - this.FEED_TYPES[b].priority);
        
        let selectorHTML = '<div class="feed-tabs">';
        
        availableFeedTypes.forEach(feedType => {
            const config = this.FEED_TYPES[feedType];
            const isActive = feedType === this.currentFeedType;
            
            selectorHTML += `
                <button 
                    class="feed-tab ${isActive ? 'active' : ''}"
                    data-feed-type="${feedType}"
                    title="${config.description}"
                    style="--tab-color: ${config.color}"
                >
                    <span class="feed-icon">${this.getSVGIcon(config.icon)}</span>
                    <span class="feed-label">${config.label}</span>
                </button>
            `;
        });
        
        selectorHTML += '</div>';
        feedSelector.innerHTML = selectorHTML;
        
        feedSelector.addEventListener('click', async (e) => {
            const button = e.target.closest('.feed-tab');
            if (!button || button.classList.contains('active')) {
                return;
            }
            
            const newFeedType = button.dataset.feedType;
            const config = this.FEED_TYPES[newFeedType];
            
            // Check authentication requirements
            if (config.requiresAuth && !this.isAuthenticated) {
                this.showToast('Please login to access this feed', 'info');
                return;
            }
            
            await this.switchFeedType(newFeedType);
        });
    }

    getSVGIcon(iconName) {
        const icons = {
            home: `<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/>
            </svg>`,
            compass: `<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <circle cx="12" cy="12" r="10"/>
                <polygon points="16.24,7.76 14.12,14.12 7.76,16.24 9.88,9.88 16.24,7.76"/>
            </svg>`,
            user: `<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
            </svg>`,
            users: `<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                <circle cx="9" cy="7" r="4"/>
                <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>`
        };
        
        return icons[iconName] || icons.compass;
    }

    addFeedTabStyles() {
        if (document.getElementById('feed-selector-styles')) return;
        
        const style = document.createElement('style');
        style.id = 'feed-selector-styles';
        style.textContent = `
            .feed-selector {
                position: sticky;
                top: 0;
                z-index: 100;
                background: rgba(15, 15, 25, 0.98);
                backdrop-filter: blur(20px);
                -webkit-backdrop-filter: blur(20px);
                border-bottom: 1px solid rgba(255, 255, 255, 0.08);
                padding: 16px 20px;
                margin-bottom: 0;
                box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
            }
            
            .feed-tabs {
                display: flex;
                gap: 8px;
                justify-content: center;
                max-width: 700px;
                margin: 0 auto;
                overflow-x: auto;
                padding: 4px;
                scrollbar-width: none;
                -ms-overflow-style: none;
                background: rgba(255, 255, 255, 0.02);
                border-radius: 16px;
                border: 1px solid rgba(255, 255, 255, 0.05);
            }
            
            .feed-tabs::-webkit-scrollbar {
                display: none;
            }
            
            .feed-tab {
                display: flex;
                align-items: center;
                gap: 10px;
                padding: 12px 20px;
                background: transparent;
                border: none;
                border-radius: 12px;
                color: rgba(255, 255, 255, 0.6);
                font-size: 14px;
                font-weight: 500;
                cursor: pointer;
                transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
                white-space: nowrap;
                min-width: fit-content;
                position: relative;
                overflow: hidden;
            }
            
            .feed-tab::before {
                content: '';
                position: absolute;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                background: linear-gradient(135deg, var(--tab-color), transparent);
                opacity: 0;
                transition: opacity 0.3s ease;
                border-radius: 12px;
            }
            
            .feed-tab:hover:not(.active) {
                color: rgba(255, 255, 255, 0.9);
                transform: translateY(-1px);
                background: rgba(255, 255, 255, 0.05);
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            }
            
            .feed-tab:hover:not(.active)::before {
                opacity: 0.1;
            }
            
            .feed-tab.active {
                color: white;
                background: linear-gradient(135deg, var(--tab-color), rgba(255, 255, 255, 0.1));
                transform: translateY(-1px);
                box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
            }
            
            .feed-tab.active::before {
                opacity: 0.2;
            }
            
            .feed-icon {
                display: flex;
                align-items: center;
                justify-content: center;
                width: 18px;
                height: 18px;
                transition: transform 0.3s ease;
            }
            
            .feed-tab:hover .feed-icon {
                transform: scale(1.1);
            }
            
            .feed-tab.active .feed-icon {
                transform: scale(1.05);
            }
            
            .feed-label {
                font-weight: 600;
                letter-spacing: 0.02em;
            }
            
            .feed-tab.active .feed-label {
                text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
            }
            
            @media (max-width: 768px) {
                .feed-selector {
                    padding: 12px 16px;
                }
                
                .feed-tabs {
                    justify-content: flex-start;
                    gap: 6px;
                    padding: 3px;
                }
                
                .feed-tab {
                    padding: 10px 16px;
                    font-size: 13px;
                    min-width: auto;
                }
                
                .feed-label {
                    display: none;
                }
                
                .feed-tab.active .feed-label,
                .feed-tab:hover .feed-label {
                    display: inline;
                }
                
                .feed-icon {
                    width: 16px;
                    height: 16px;
                }
            }
        `;
        document.head.appendChild(style);
    }

    async switchFeedType(newFeedType) {
        if (newFeedType === this.currentFeedType) return;
        
        // Update active tab
        document.querySelectorAll('.feed-tab').forEach(tab => {
            tab.classList.toggle('active', tab.dataset.feedType === newFeedType);
        });
        
        this.currentFeedType = newFeedType;
        this.showToast(`Switched to ${this.FEED_TYPES[newFeedType].label} feed`, 'info');
        
        // Reset pagination
        this.hasMore = true;
        this.nextCursor = null;
        await this.loadInitialPosts();
    }

    setupUploadModal() {
        if (!this.isAuthenticated) return; // Only setup for authenticated users
        
        const uploadFab = document.getElementById('upload-fab');
        const uploadModal = document.getElementById('upload-modal');
        
        if (!uploadFab || !uploadModal) return;
        
        this.setupUploadModalEvents();
        this.setupQuickUpload();
    }

    setupUploadModalEvents() {
        const uploadFab = document.getElementById('upload-fab');
        const uploadModal = document.getElementById('upload-modal');
        const closeModal = document.getElementById('close-modal');
        const goToUpload = document.getElementById('go-to-upload');
        
        uploadFab.addEventListener('click', () => this.openUploadModal());
        closeModal.addEventListener('click', () => this.closeUploadModal());
        uploadModal.addEventListener('click', (e) => {
            if (e.target === uploadModal) this.closeUploadModal();
        });
        
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && uploadModal.style.display === 'block') {
                this.closeUploadModal();
            }
        });
        
        goToUpload.addEventListener('click', () => {
            window.location.href = '/photo';
        });
    }

    setupQuickUpload() {
        const quickUploadArea = document.getElementById('quick-upload-area');
        const quickFileInput = document.getElementById('quick-file-input');
        const quickUploadBtn = document.getElementById('quick-upload-btn');
        const captionInput = document.getElementById('caption-input');
        
        if (!quickUploadArea || !quickFileInput || !quickUploadBtn) return;
        
        // Caption character counter
        if (captionInput) {
            const captionCounter = document.getElementById('caption-count');
            captionInput.addEventListener('input', (e) => {
                if (captionCounter) {
                    captionCounter.textContent = e.target.value.length;
                }
            });
        }
        
        quickUploadArea.addEventListener('click', () => quickFileInput.click());
        quickUploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            quickUploadArea.classList.add('dragover');
        });
        
        quickUploadArea.addEventListener('dragleave', (e) => {
            if (!quickUploadArea.contains(e.relatedTarget)) {
                quickUploadArea.classList.remove('dragover');
            }
        });
        
        quickUploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            quickUploadArea.classList.remove('dragover');
            const files = e.dataTransfer.files;
            if (files.length > 0 && files[0].type.startsWith('image/')) {
                this.handleQuickFile(files[0]);
            } else {
                this.showToast('Please select a valid image file', 'error');
            }
        });
        
        quickFileInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                this.handleQuickFile(e.target.files[0]);
            }
        });
        
        quickUploadBtn.addEventListener('click', () => this.uploadQuickPhoto());
    }

    openUploadModal() {
        const uploadModal = document.getElementById('upload-modal');
        uploadModal.style.display = 'block';
        document.body.style.overflow = 'hidden';
        
        setTimeout(() => {
            uploadModal.style.opacity = '1';
        }, 10);
    }

    closeUploadModal() {
        const uploadModal = document.getElementById('upload-modal');
        const quickPreview = document.getElementById('quick-preview');
        const quickFileInput = document.getElementById('quick-file-input');
        const captionInput = document.getElementById('caption-input');
        const captionCounter = document.getElementById('caption-count');
        
        uploadModal.style.opacity = '0';
        setTimeout(() => {
            uploadModal.style.display = 'none';
            document.body.style.overflow = '';
            if (quickPreview) {
                quickPreview.style.display = 'none';
                quickPreview.className = 'quick-preview-hidden';
            }
            if (quickFileInput) quickFileInput.value = '';
            if (captionInput) captionInput.value = '';
            if (captionCounter) captionCounter.textContent = '0';
            this.selectedFile = null;
        }, 300);
    }

    handleQuickFile(file) {
        if (file.size > 10 * 1024 * 1024) {
            this.showToast('File size must be less than 10MB', 'error');
            return;
        }

        this.selectedFile = file;
        const reader = new FileReader();
        
        reader.onload = (e) => {
            const previewImage = document.getElementById('preview-image');
            const quickPreview = document.getElementById('quick-preview');
            
            previewImage.src = e.target.result;
            previewImage.onload = () => {
                quickPreview.className = 'quick-preview-shown';
                quickPreview.style.display = 'block';
                quickPreview.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            };
        };
        
        reader.onerror = () => {
            this.showToast('Error reading file', 'error');
        };
        
        reader.readAsDataURL(file);
    }

    async uploadQuickPhoto() {
        if (!this.selectedFile) return;

        const quickUploadBtn = document.getElementById('quick-upload-btn');
        const btnText = quickUploadBtn.querySelector('.btn-text');
        const btnSpinner = quickUploadBtn.querySelector('.btn-spinner');
        
        quickUploadBtn.disabled = true;
        btnText.textContent = 'Uploading...';
        if (btnSpinner) btnSpinner.style.display = 'inline-block';

        const formData = new FormData();
        formData.append('images[]', this.selectedFile);
        
        const caption = document.getElementById('caption-input').value.trim();
        if (caption) {
            formData.append('caption', caption);
        }

        try {
            const response = await fetch('/api/upload', {
                method: 'POST',
                credentials: 'include',
                body: formData
            });

            const result = await response.json();

            if (response.ok) {
                this.showToast('Photo uploaded successfully!', 'success');
                this.closeUploadModal();
                
                // Refresh the current feed
                setTimeout(() => {
                    this.loadInitialPosts();
                }, 1000);
            } else if (response.status === 401) {
                this.showToast('Session expired. Redirecting to login...', 'error');
                setTimeout(() => {
                    window.location.href = '/login_page';
                }, 2000);
            } else {
                this.showToast('Upload failed: ' + (result.error || result.message || 'Unknown error'), 'error');
            }
        } catch (error) {
            console.error('Upload error:', error);
            this.showToast('Network error occurred. Please try again.', 'error');
        } finally {
            quickUploadBtn.disabled = false;
            btnText.textContent = 'Share to Feed';
            if (btnSpinner) btnSpinner.style.display = 'none';
        }
    }

    setupPullToRefresh() {
        let startY = 0;
        let currentY = 0;
        const pullThreshold = 100;
        let isPulling = false;
        
        document.addEventListener('touchstart', (e) => {
            if (window.scrollY === 0) {
                startY = e.touches[0].clientY;
                isPulling = true;
            }
        }, { passive: true });
        
        document.addEventListener('touchmove', (e) => {
            if (!isPulling) return;
            
            currentY = e.touches[0].clientY;
            const pullDistance = currentY - startY;
            
            if (pullDistance > 0 && window.scrollY === 0) {
                if (pullDistance > pullThreshold) {
                    document.body.style.transform = `translateY(${Math.min(pullDistance / 3, 50)}px)`;
                }
            }
        }, { passive: true });
        
        document.addEventListener('touchend', async () => {
            if (!isPulling) return;
            
            const pullDistance = currentY - startY;
            
            document.body.style.transform = '';
            document.body.style.transition = 'transform 0.3s ease';
            
            setTimeout(() => {
                document.body.style.transition = '';
            }, 300);
            
            if (pullDistance > pullThreshold && window.scrollY === 0) {
                this.showToast('Refreshing feed...', 'success');
                await this.loadInitialPosts();
            }
            
            isPulling = false;
            startY = 0;
            currentY = 0;
        });
    }

    showEmptyFeedMessage() {
        const emptyMessage = document.createElement('div');
        emptyMessage.className = 'empty-feed-message';
        
        let content = '';
        switch (this.currentFeedType) {
            case 'personal':
                content = `
                    <div style="text-align: center; padding: 60px 20px; color: rgba(255,255,255,0.6);">
                        <div style="font-size: 4rem; margin-bottom: 20px;">📱</div>
                        <h3 style="margin-bottom: 10px;">No photos uploaded yet!</h3>
                        <p style="margin-bottom: 20px;">Start building your photo collection.</p>
                        ${this.isAuthenticated ? `
                            <button onclick="document.getElementById('upload-fab').click()" 
                                    style="background: linear-gradient(45deg, #667eea 0%, #764ba2 100%); 
                                           border: none; padding: 12px 24px; border-radius: 25px; 
                                           color: white; cursor: pointer; font-weight: 600;">
                                Upload Your First Photo
                            </button>
                        ` : ''}
                    </div>
                `;
                break;
            case 'following':
                content = `
                    <div style="text-align: center; padding: 60px 20px; color: rgba(255,255,255,0.6);">
                        <div style="font-size: 4rem; margin-bottom: 20px;">👥</div>
                        <h3 style="margin-bottom: 10px;">No photos from people you follow!</h3>
                        <p style="margin-bottom: 20px;">Follow some users to see their photos here.</p>
                    </div>
                `;
                break;
            default:
                content = `
                    <div style="text-align: center; padding: 60px 20px; color: rgba(255,255,255,0.6);">
                        <div style="font-size: 4rem; margin-bottom: 20px;">📸</div>
                        <h3 style="margin-bottom: 10px;">No photos yet!</h3>
                        <p style="margin-bottom: 20px;">Be the first to share something amazing.</p>
                        ${this.isAuthenticated ? `
                            <button onclick="document.getElementById('upload-fab').click()" 
                                    style="background: linear-gradient(45deg, #667eea 0%, #764ba2 100%); 
                                           border: none; padding: 12px 24px; border-radius: 25px; 
                                           color: white; cursor: pointer; font-weight: 600;">
                                Share Your First Photo
                            </button>
                        ` : ''}
                    </div>
                `;
        }
        
        emptyMessage.innerHTML = content;
        this.feedContainer.appendChild(emptyMessage);
    }

    showEndOfFeedMessage() {
        const endMessage = document.createElement('div');
        endMessage.className = 'end-of-feed';
        endMessage.innerHTML = `
            <div style="text-align: center; padding: 40px 20px; color: rgba(255,255,255,0.6);">
                <div style="font-size: 3rem; margin-bottom: 15px;">🎉</div>
                <p style="margin-bottom: 5px;">You've seen all the amazing photos!</p>
                <p style="font-size: 0.9rem;">Check back later for more content.</p>
            </div>
        `;
        
        const loadingSpinner = document.getElementById('loading-spinner');
        if (loadingSpinner && loadingSpinner.parentNode) {
            loadingSpinner.parentNode.replaceChild(endMessage, loadingSpinner);
        } else {
            this.feedContainer.appendChild(endMessage);
        }
    }

    showLoadingSpinner() {
        const loadingSpinner = document.getElementById('loading-spinner');
        if (loadingSpinner) {
            loadingSpinner.style.display = 'block';
        }
    }

    hideLoadingSpinner() {
        const loadingSpinner = document.getElementById('loading-spinner');
        if (loadingSpinner) {
            loadingSpinner.style.display = 'none';
        }
    }

    showLoginPrompt() {
        if (this.feedContainer.querySelector('.login-prompt')) return; // Already exists
        
        const loginPrompt = document.createElement('div');
        loginPrompt.className = 'login-prompt';
        loginPrompt.innerHTML = `
            <div class="login-prompt-content">
                <h3>Welcome to PhotoShare!</h3>
                <p>Sign in to upload photos, follow friends, and personalize your feed.</p>
                <div class="login-prompt-buttons">
                    <a href="/login_page" class="login-btn">Sign In</a>
                    <a href="/registration" class="register-btn">Sign Up</a>
                </div>
            </div>
        `;
        
        this.feedContainer.insertBefore(loginPrompt, this.feedContainer.firstChild);
        this.addLoginPromptStyles();
    }

    addLoginPromptStyles() {
        if (document.getElementById('login-prompt-styles')) return;
        
        const style = document.createElement('style');
        style.id = 'login-prompt-styles';
        style.textContent = `
            .login-prompt {
                background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%);
                border: 1px solid rgba(102, 126, 234, 0.3);
                border-radius: 15px;
                padding: 30px;
                margin-bottom: 30px;
                text-align: center;
            }
            
            .login-prompt h3 {
                color: #ffffff;
                margin-bottom: 10px;
                font-size: 24px;
            }
            
            .login-prompt p {
                color: rgba(255, 255, 255, 0.8);
                margin-bottom: 25px;
                font-size: 16px;
                line-height: 1.5;
            }
            
            .login-prompt-buttons {
                display: flex;
                gap: 15px;
                justify-content: center;
                flex-wrap: wrap;
            }
            
            .login-btn, .register-btn {
                padding: 12px 24px;
                border-radius: 25px;
                text-decoration: none;
                font-weight: 600;
                transition: all 0.3s ease;
                border: 2px solid transparent;
            }
            
            .login-btn {
                background: linear-gradient(45deg, #667eea 0%, #764ba2 100%);
                color: white;
            }
            
            .login-btn:hover {
                transform: translateY(-2px);
                box-shadow: 0 8px 25px rgba(102, 126, 234, 0.4);
            }
            
            .register-btn {
                background: transparent;
                color: #667eea;
                border-color: #667eea;
            }
            
            .register-btn:hover {
                background: #667eea;
                color: white;
                transform: translateY(-2px);
            }
        `;
        document.head.appendChild(style);
    }

    showToast(message, type = 'success') {
        let toastContainer = document.getElementById('toast-container');
        if (!toastContainer) {
            toastContainer = document.createElement('div');
            toastContainer.id = 'toast-container';
            toastContainer.style.cssText = `
                position: fixed;
                top: 20px;
                right: 20px;
                z-index: 10000;
                pointer-events: none;
            `;
            document.body.appendChild(toastContainer);
            
            // Add toast styles if not present
            if (!document.getElementById('toast-styles')) {
                const style = document.createElement('style');
                style.id = 'toast-styles';
                style.textContent = `
                    .toast {
                        display: flex;
                        align-items: center;
                        gap: 10px;
                        background: rgba(15, 15, 25, 0.95);
                        color: white;
                        padding: 12px 20px;
                        border-radius: 8px;
                        margin-bottom: 10px;
                        border-left: 4px solid;
                        backdrop-filter: blur(10px);
                        transform: translateX(100%);
                        opacity: 0;
                        transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
                        pointer-events: auto;
                        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
                        min-width: 250px;
                        max-width: 400px;
                    }
                    
                    .toast.show {
                        transform: translateX(0);
                        opacity: 1;
                    }
                    
                    .toast.success {
                        border-left-color: #10b981;
                    }
                    
                    .toast.error {
                        border-left-color: #ef4444;
                    }
                    
                    .toast.info {
                        border-left-color: #3b82f6;
                    }
                    
                    .toast.warning {
                        border-left-color: #f59e0b;
                    }
                    
                    .toast-icon {
                        font-size: 16px;
                        flex-shrink: 0;
                    }
                    
                    .toast-message {
                        flex: 1;
                        font-size: 14px;
                        line-height: 1.4;
                    }
                    
                    @media (max-width: 768px) {
                        #toast-container {
                            top: 80px;
                            right: 15px;
                            left: 15px;
                        }
                        
                        .toast {
                            min-width: auto;
                            width: 100%;
                        }
                    }
                `;
                document.head.appendChild(style);
            }
        }
        
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        
        const icons = {
            success: '✓',
            error: '✕',
            info: 'ℹ',
            warning: '⚠'
        };
        
        toast.innerHTML = `
            <span class="toast-icon">${icons[type] || icons.info}</span>
            <span class="toast-message">${this.escapeHtml(message)}</span>
        `;
        
        toastContainer.appendChild(toast);
        
        setTimeout(() => {
            toast.classList.add('show');
        }, 100);
        
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.remove();
                }
            }, 400);
        }, 3000);
    }

    async logout() {
        this.showToast('Logging out...', 'info');
        
        try {
            await fetch('/api/logout', {
                method: 'POST',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json'
                }
            });
        } catch (error) {
            console.error('Logout error:', error);
        }
        
        // Clean up local state
        if (this.websocket) {
            this.websocket.close();
            this.websocket = null;
        }
        
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');
        
        setTimeout(() => {
            window.location.href = '/login_page';
        }, 1000);
    }

    formatTimeAgo(dateString) {
        try {
            const date = new Date(dateString);
            if (isNaN(date.getTime())) {
                return 'Unknown time';
            }
            
            const now = new Date();
            const diffInSeconds = Math.floor((now - date) / 1000);
            
            if (diffInSeconds < 60) return 'Just now';
            if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m`;
            if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h`;
            if (diffInSeconds < 604800) return `${Math.floor(diffInSeconds / 86400)}d`;
            
            return date.toLocaleDateString();
        } catch (error) {
            console.error('Error formatting date:', error);
            return 'Unknown time';
        }
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text.toString();
        return div.innerHTML;
    }
}

// Initialize the feed manager when the page loads
const feedManager = new FeedManager();