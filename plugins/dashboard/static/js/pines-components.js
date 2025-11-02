/**
 * Pines UI Components for AuthSome Dashboard
 * Alpine.js + Tailwind CSS component library
 */

/**
 * Modal Component
 * Usage: x-data="modal()"
 */
function modal(initialOpen = false) {
    return {
        open: initialOpen,
        
        show() {
            this.open = true;
            document.body.style.overflow = 'hidden';
        },
        
        close() {
            this.open = false;
            document.body.style.overflow = '';
        },
        
        toggle() {
            this.open ? this.close() : this.show();
        }
    };
}

/**
 * Dropdown Component
 * Usage: x-data="dropdown()"
 */
function dropdown(options = {}) {
    return {
        open: false,
        placement: options.placement || 'bottom-end',
        
        toggle() {
            this.open = !this.open;
        },
        
        close() {
            this.open = false;
        },
        
        handleClickAway(event) {
            if (!this.$el.contains(event.target)) {
                this.close();
            }
        }
    };
}

/**
 * Notification/Toast Component
 * Usage: x-data="notification()"
 */
function notification() {
    return {
        notifications: [],
        
        show(message, type = 'info', duration = 3000) {
            const id = Date.now();
            this.notifications.push({ id, message, type });
            
            if (duration > 0) {
                setTimeout(() => this.remove(id), duration);
            }
        },
        
        remove(id) {
            this.notifications = this.notifications.filter(n => n.id !== id);
        },
        
        success(message, duration = 3000) {
            this.show(message, 'success', duration);
        },
        
        error(message, duration = 4000) {
            this.show(message, 'error', duration);
        },
        
        warning(message, duration = 3500) {
            this.show(message, 'warning', duration);
        },
        
        info(message, duration = 3000) {
            this.show(message, 'info', duration);
        }
    };
}

/**
 * Tabs Component
 * Usage: x-data="tabs(['tab1', 'tab2'])"
 */
function tabs(tabList = [], initialTab = 0) {
    return {
        tabs: tabList,
        activeTab: initialTab,
        
        isActive(index) {
            return this.activeTab === index;
        },
        
        setActive(index) {
            this.activeTab = index;
        }
    };
}

/**
 * Accordion Component
 * Usage: x-data="accordion()"
 */
function accordion(allowMultiple = false) {
    return {
        openItems: [],
        allowMultiple,
        
        toggle(id) {
            if (this.isOpen(id)) {
                this.openItems = this.openItems.filter(item => item !== id);
            } else {
                if (!this.allowMultiple) {
                    this.openItems = [id];
                } else {
                    this.openItems.push(id);
                }
            }
        },
        
        isOpen(id) {
            return this.openItems.includes(id);
        },
        
        close(id) {
            this.openItems = this.openItems.filter(item => item !== id);
        },
        
        closeAll() {
            this.openItems = [];
        }
    };
}

/**
 * Sidebar Component
 * Usage: x-data="sidebar()"
 */
function sidebar(initialOpen = true) {
    return {
        open: initialOpen,
        mobile: window.innerWidth < 1024,
        
        init() {
            // Listen for window resize
            window.addEventListener('resize', () => {
                this.mobile = window.innerWidth < 1024;
                if (!this.mobile) {
                    this.open = true;
                }
            });
        },
        
        toggle() {
            this.open = !this.open;
        },
        
        close() {
            this.open = false;
        }
    };
}

/**
 * Confirm Dialog Component
 * Usage: x-data="confirmDialog()"
 */
function confirmDialog() {
    return {
        show: false,
        title: '',
        message: '',
        confirmText: 'Confirm',
        cancelText: 'Cancel',
        confirmCallback: null,
        cancelCallback: null,
        
        confirm(options) {
            this.title = options.title || 'Confirm';
            this.message = options.message || 'Are you sure?';
            this.confirmText = options.confirmText || 'Confirm';
            this.cancelText = options.cancelText || 'Cancel';
            this.confirmCallback = options.onConfirm || null;
            this.cancelCallback = options.onCancel || null;
            this.show = true;
        },
        
        handleConfirm() {
            if (this.confirmCallback) {
                this.confirmCallback();
            }
            this.close();
        },
        
        handleCancel() {
            if (this.cancelCallback) {
                this.cancelCallback();
            }
            this.close();
        },
        
        close() {
            this.show = false;
            this.confirmCallback = null;
            this.cancelCallback = null;
        }
    };
}

/**
 * Table Component with Sorting & Pagination
 * Usage: x-data="table(data, columns)"
 */
function table(data = [], columns = []) {
    return {
        data: data,
        columns: columns,
        sortColumn: null,
        sortDirection: 'asc',
        currentPage: 1,
        pageSize: 10,
        searchQuery: '',
        
        get filteredData() {
            let result = this.data;
            
            // Apply search filter
            if (this.searchQuery) {
                result = result.filter(item => {
                    return Object.values(item).some(value => 
                        String(value).toLowerCase().includes(this.searchQuery.toLowerCase())
                    );
                });
            }
            
            // Apply sorting
            if (this.sortColumn) {
                result = [...result].sort((a, b) => {
                    const aVal = a[this.sortColumn];
                    const bVal = b[this.sortColumn];
                    
                    if (aVal < bVal) return this.sortDirection === 'asc' ? -1 : 1;
                    if (aVal > bVal) return this.sortDirection === 'asc' ? 1 : -1;
                    return 0;
                });
            }
            
            return result;
        },
        
        get paginatedData() {
            const start = (this.currentPage - 1) * this.pageSize;
            const end = start + this.pageSize;
            return this.filteredData.slice(start, end);
        },
        
        get totalPages() {
            return Math.ceil(this.filteredData.length / this.pageSize);
        },
        
        sort(column) {
            if (this.sortColumn === column) {
                this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc';
            } else {
                this.sortColumn = column;
                this.sortDirection = 'asc';
            }
        },
        
        nextPage() {
            if (this.currentPage < this.totalPages) {
                this.currentPage++;
            }
        },
        
        prevPage() {
            if (this.currentPage > 1) {
                this.currentPage--;
            }
        },
        
        goToPage(page) {
            if (page >= 1 && page <= this.totalPages) {
                this.currentPage = page;
            }
        }
    };
}

/**
 * Form Validation Component
 * Usage: x-data="formValidation(rules)"
 */
function formValidation(rules = {}) {
    return {
        errors: {},
        touched: {},
        
        validate(field, value) {
            if (!rules[field]) return true;
            
            const fieldRules = rules[field];
            this.errors[field] = null;
            
            // Required validation
            if (fieldRules.required && !value) {
                this.errors[field] = fieldRules.requiredMessage || 'This field is required';
                return false;
            }
            
            // Min length validation
            if (fieldRules.minLength && value.length < fieldRules.minLength) {
                this.errors[field] = fieldRules.minLengthMessage || `Minimum ${fieldRules.minLength} characters required`;
                return false;
            }
            
            // Max length validation
            if (fieldRules.maxLength && value.length > fieldRules.maxLength) {
                this.errors[field] = fieldRules.maxLengthMessage || `Maximum ${fieldRules.maxLength} characters allowed`;
                return false;
            }
            
            // Email validation
            if (fieldRules.email) {
                const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                if (!emailRegex.test(value)) {
                    this.errors[field] = fieldRules.emailMessage || 'Invalid email address';
                    return false;
                }
            }
            
            // Custom validation
            if (fieldRules.custom && !fieldRules.custom(value)) {
                this.errors[field] = fieldRules.customMessage || 'Invalid value';
                return false;
            }
            
            return true;
        },
        
        touch(field) {
            this.touched[field] = true;
        },
        
        hasError(field) {
            return this.touched[field] && this.errors[field];
        },
        
        getError(field) {
            return this.errors[field];
        },
        
        isValid() {
            return Object.keys(this.errors).every(key => !this.errors[key]);
        },
        
        reset() {
            this.errors = {};
            this.touched = {};
        }
    };
}

/**
 * Loading State Component
 * Usage: x-data="loading()"
 */
function loading(initialState = false) {
    return {
        isLoading: initialState,
        
        start() {
            this.isLoading = true;
        },
        
        stop() {
            this.isLoading = false;
        },
        
        async wrap(promise) {
            this.start();
            try {
                const result = await promise;
                return result;
            } finally {
                this.stop();
            }
        }
    };
}

/**
 * Tooltip Component
 * Usage: x-data="tooltip('Tooltip text')"
 */
function tooltip(text, placement = 'top') {
    return {
        show: false,
        text: text,
        placement: placement,
        
        mouseEnter() {
            this.show = true;
        },
        
        mouseLeave() {
            this.show = false;
        }
    };
}

/**
 * Slide Over Component
 * Usage: x-data="slideOver()"
 */
function slideOver(side = 'right') {
    return {
        open: false,
        side: side,
        
        show() {
            this.open = true;
            document.body.style.overflow = 'hidden';
        },
        
        close() {
            this.open = false;
            document.body.style.overflow = '';
        }
    };
}

// Export all components for global use
window.pinesComponents = {
    modal,
    dropdown,
    notification,
    tabs,
    accordion,
    sidebar,
    confirmDialog,
    table,
    formValidation,
    loading,
    tooltip,
    slideOver
};

console.log('✨ Pines UI Components loaded');

