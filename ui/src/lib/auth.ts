import { writable } from 'svelte/store';

export const authState = writable({
    isLoading: true,
    needsSetup: false,
    isAuthenticated: false
});

export async function checkAuthStatus() {
    try {
        const res = await fetch('/api/auth/status');
        if (res.ok) {
            const data = await res.json();
            authState.set({
                isLoading: false,
                needsSetup: data.needs_setup,
                isAuthenticated: data.authenticated
            });
            return data;
        }
    } catch (e) {
        console.error("Failed to check auth status", e);
    }
    
    authState.set({
        isLoading: false,
        needsSetup: false,
        isAuthenticated: false
    });
    return null;
}
