/**
 * AuthSome Dashboard JavaScript
 * Alpine.js components and utilities
 */

/**
 * Theme Management
 * Must be defined before Alpine initializes
 */
function themeData() {
    return {
        isDark: false,
        
        initTheme() {
            // Check localStorage first
            const stored = localStorage.getItem('theme');
            if (stored) {
                this.isDark = stored === 'dark';
            } else {
                // Fall back to system preference
                this.isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            }
            
            // Apply theme
            this.updateTheme();
            
            // Listen for system theme changes
            window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
                if (!localStorage.getItem('theme')) {
                    this.isDark = e.matches;
                    this.updateTheme();
                }
            });
        },
        
        toggleTheme() {
            this.isDark = !this.isDark;
            this.updateTheme();
            localStorage.setItem('theme', this.isDark ? 'dark' : 'light');
        },
        
        updateTheme() {
            if (this.isDark) {
                document.documentElement.classList.add('dark');
            } else {
                document.documentElement.classList.remove('dark');
            }
        }
    };
}

// Initialize when Alpine is ready
document.addEventListener('alpine:init', () => {
    // Toast notification component
    Alpine.data('toast', () => ({
        show: false,
        message: '',
        type: 'info',
        
        showToast(message, type = 'info', duration = 3000) {
            this.message = message;
            this.type = type;
            this.show = true;
            
            setTimeout(() => {
                this.show = false;
            }, duration);
        },
        
        hideToast() {
            this.show = false;
        }
    }));
    
    // Modal component
    Alpine.data('modal', () => ({
        open: false,
        
        openModal() {
            this.open = true;
            document.body.style.overflow = 'hidden';
        },
        
        closeModal() {
            this.open = false;
            document.body.style.overflow = '';
        }
    }));
    
    // Confirmation dialog component
    Alpine.data('confirm', () => ({
        show: false,
        title: '',
        message: '',
        callback: null,
        
        confirm(title, message, callback) {
            this.title = title;
            this.message = message;
            this.callback = callback;
            this.show = true;
        },
        
        accept() {
            if (this.callback) {
                this.callback();
            }
            this.show = false;
        },
        
        cancel() {
            this.show = false;
            this.callback = null;
        }
    }));
    
    // Table filter component
    Alpine.data('tableFilter', () => ({
        search: '',
        
        init() {
            // Initialize search from URL parameter
            const params = new URLSearchParams(window.location.search);
            this.search = params.get('search') || '';
        },
        
        filter() {
            // Update URL with search parameter
            const url = new URL(window.location);
            if (this.search) {
                url.searchParams.set('search', this.search);
            } else {
                url.searchParams.delete('search');
            }
            window.history.pushState({}, '', url);
        }
    }));
});

/**
 * Utility Functions
 */

// Format date helper
function formatDate(dateString) {
    const date = new Date(dateString);
    return new Intl.DateTimeFormat('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    }).format(date);
}

// Format relative time
function formatTimeAgo(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const seconds = Math.floor((now - date) / 1000);
    
    const intervals = {
        year: 31536000,
        month: 2592000,
        week: 604800,
        day: 86400,
        hour: 3600,
        minute: 60
    };
    
    for (const [unit, secondsInUnit] of Object.entries(intervals)) {
        const interval = Math.floor(seconds / secondsInUnit);
        if (interval >= 1) {
            return `${interval} ${unit}${interval > 1 ? 's' : ''} ago`;
        }
    }
    
    return 'just now';
}

// Copy to clipboard
async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        showToast('Copied to clipboard', 'success');
    } catch (err) {
        showToast('Failed to copy', 'error');
    }
}

// Show toast notification
function showToast(message, type = 'info', duration = 3000) {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type} rounded-lg p-4 shadow-lg`;
    toast.textContent = message;
    toast.style.animation = 'fadeIn 0.3s ease-out';
    
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.style.animation = 'fadeOut 0.3s ease-out';
        setTimeout(() => {
            document.body.removeChild(toast);
        }, 300);
    }, duration);
}

// Confirm action
function confirmAction(message, callback) {
    if (confirm(message)) {
        callback();
    }
}

// Debounce function
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Throttle function
function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// AJAX helper
async function fetchJSON(url, options = {}) {
    const defaultOptions = {
        headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
        }
    };
    
    try {
        const response = await fetch(url, { ...defaultOptions, ...options });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        return await response.json();
    } catch (error) {
        console.error('Fetch error:', error);
        throw error;
    }
}

// Form submission helper
async function submitForm(formElement, onSuccess, onError) {
    const formData = new FormData(formElement);
    const data = Object.fromEntries(formData.entries());
    
    try {
        const response = await fetchJSON(formElement.action, {
            method: formElement.method || 'POST',
            body: JSON.stringify(data)
        });
        
        if (onSuccess) {
            onSuccess(response);
        }
    } catch (error) {
        if (onError) {
            onError(error);
        } else {
            showToast('Form submission failed', 'error');
        }
    }
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    // Ctrl/Cmd + K: Focus search
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        const searchInput = document.querySelector('#search');
        if (searchInput) {
            searchInput.focus();
        }
    }
    
    // Escape: Close modals
    if (e.key === 'Escape') {
        // Close any open modals
        document.querySelectorAll('[x-data]').forEach(el => {
            if (el._x_dataStack && el._x_dataStack[0].open) {
                el._x_dataStack[0].open = false;
            }
        });
    }
});

// Auto-refresh for dashboard stats (optional)
let refreshInterval = null;

function startAutoRefresh(intervalSeconds = 30) {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
    
    refreshInterval = setInterval(() => {
        // Only refresh if on dashboard page
        if (window.location.pathname === '/dashboard/') {
            console.log('Auto-refreshing dashboard stats...');
            // In a real implementation, this would fetch updated stats via AJAX
        }
    }, intervalSeconds * 1000);
}

function stopAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
}

// Clean up on page unload
window.addEventListener('beforeunload', () => {
    stopAutoRefresh();
});

// Export utilities for global use
window.dashboardUtils = {
    formatDate,
    formatTimeAgo,
    copyToClipboard,
    showToast,
    confirmAction,
    debounce,
    throttle,
    fetchJSON,
    submitForm,
    startAutoRefresh,
    stopAutoRefresh
};

console.log('AuthSome Dashboard initialized');

