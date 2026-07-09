import { defineStore } from "pinia";
import { ref } from "vue";
import { useAuthStore } from "./auth_store.ts";
import type {
  WsEvent,
  WsEventType,
  WsHandler,
} from "../../models/ws_models.ts";

const BASE_BACKOFF_MS = 1000;
const MAX_BACKOFF_MS = 30000;
const MAX_ATTEMPTS = 10;

export const useWsStore = defineStore("ws", () => {
  const connected = ref(false);

  const handlers = new Map<WsEventType, Set<WsHandler>>();
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;
  let attempt = 0;
  let intentionalClose = false;

  const endpoint = (): string => {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${window.location.host}/api/ws`;
  };

  const on = (type: WsEventType, handler: WsHandler): (() => void) => {
    const existing = handlers.get(type) ?? new Set<WsHandler>();
    existing.add(handler);
    handlers.set(type, existing);
    return () => {
      handlers.get(type)?.delete(handler);
    };
  };

  const dispatch = (raw: string): void => {
    let event: WsEvent;
    try {
      event = JSON.parse(raw) as WsEvent;
    } catch {
      return;
    }
    handlers.get(event.type)?.forEach((handler) => handler(event.payload));
  };

  const scheduleReconnect = (): void => {
    // The handshake's 401 is invisible to the WebSocket API, so an expired session
    // can only be bounded by the attempt cap.
    const authStore = useAuthStore();
    if (!authStore.isAuthenticated || attempt >= MAX_ATTEMPTS) return;

    const backoff = Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** attempt);
    const jittered = backoff * (0.5 + Math.random() * 0.5);
    attempt += 1;

    reconnectTimer = window.setTimeout(connect, jittered);
  };

  const connect = (): void => {
    if (socket) return;

    intentionalClose = false;
    socket = new WebSocket(endpoint());

    socket.onopen = () => {
      connected.value = true;
      attempt = 0;
    };
    socket.onmessage = (message: MessageEvent<string>) =>
      dispatch(message.data);
    socket.onclose = () => {
      connected.value = false;
      socket = null;
      if (!intentionalClose) scheduleReconnect();
    };
  };

  const disconnect = (): void => {
    intentionalClose = true;
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    socket?.close();
    socket = null;
    connected.value = false;
    attempt = 0;
  };

  return { connected, on, connect, disconnect };
});
