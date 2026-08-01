import { writable } from 'svelte/store';

export interface BuildLogEntry {
    timestamp: string;
    stage: string;    // "init" | "detect" | "generate" | "build" | "deploy" | "runtime"
    service: string;
    message: string;
    level: string;    // "info" | "warn" | "error" | "success"
}

export function createLogStore(projectId: string) {
    const { subscribe, set, update } = writable<BuildLogEntry[]>([]);
    let ws: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    function connect() {
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${protocol}//${window.location.host}/ws/projects/${projectId}/logs`);

        ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);

                // Server signals that a new build has started → clear the view.
                if (data.type === 'clear') {
                    set([]);
                    return;
                }

                // Normal build log entry.
                update(logs => [...logs, data as BuildLogEntry]);
            } catch {
                // Ignore unparseable frames.
            }
        };

        ws.onclose = () => {
            // Reconnect after 3 s (only when page is still open).
            reconnectTimer = setTimeout(connect, 3000);
        };

        ws.onerror = () => {
            ws?.close();
        };
    }

    function clear() {
        // Tell the backend to clear the server-side buffer (triggers clear signal to all WS clients).
        fetch(`/api/projects/${projectId}/logs`, { method: 'DELETE' });
        // Optimistically clear local store immediately.
        set([]);
    }

    function disconnect() {
        if (reconnectTimer) clearTimeout(reconnectTimer);
        ws?.close();
        ws = null;
    }

    connect();

    return {
        subscribe,
        clear,
        disconnect,
    };
}