document.addEventListener('alpine:init', () => {
    Alpine.store('app', {
        siteName: 'ResCMS',
        user: null,
        authenticated: false,
        notifications: [],
        currentTab: localStorage.getItem('res_cms_tab') || 'home',
        
        async init() {
            // Reset to home if filtering by category or tag
            const params = new URLSearchParams(window.location.search);
            if (params.has('category') || params.has('tag')) {
                this.currentTab = 'home';
            }
            await this.fetchSession();
            await this.fetchSettings();
        },
        
        setTab(tab) {
            this.currentTab = tab;
            localStorage.setItem('res_cms_tab', tab);
            if (window.location.pathname !== '/') {
                window.location.href = '/';
            }
        },
        
        async fetchSession() {
            try {
                const res = await fetch('/api/v1/session');
                const data = await res.json();
                this.authenticated = data.authenticated;
                this.user = data.user;
            } catch (err) {
                console.error('Failed to fetch session', err);
            }
        },
        
        async fetchSettings() {
            try {
                const res = await fetch('/api/v1/settings');
                const data = await res.json();
                this.siteName = data.blog_name || 'ResCMS';
            } catch (err) {
                console.error('Failed to fetch settings', err);
            }
        },
        
        notify(message, type = 'info') {
            const id = Date.now();
            this.notifications.push({ id, message, type });
            setTimeout(() => {
                this.notifications = this.notifications.filter(n => n.id !== id);
            }, 5000);
        }
    });
});
