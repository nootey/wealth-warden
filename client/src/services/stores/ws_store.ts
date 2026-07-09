import { defineStore } from "pinia";
import { ref } from "vue";
import { useAuthStore } from "./auth_store.ts";
import { useToastStore } from "./toast_store.ts";
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
  const attempts = ref(0);

  const handlers = new Map<WsEventType, Set<WsHandler>>();
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;

  // Only a hand-driven connect from settings toasts its outcome; the login
  // connect and the backoff loop stay silent.
  let announce = false;

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
    if (!authStore.isAuthenticated || attempts.value >= MAX_ATTEMPTS) return;

    const backoff = Math.min(
      MAX_BACKOFF_MS,
      BASE_BACKOFF_MS * 2 ** attempts.value,
    );
    const jittered = backoff * (0.5 + Math.random() * 0.5);
    attempts.value += 1;

    reconnectTimer = window.setTimeout(connect, jittered);
  };

  const connect = (): void => {
    if (socket) return;

    socket = new WebSocket(endpoint());

    socket.onopen = () => {
      connected.value = true;
      attempts.value = 0;
      if (announce) {
        announce = false;
        useToastStore().createInfoToast(
          "Connected",
          "Real-time event updates enabled.",
        );
      }
    };
    socket.onmessage = (message: MessageEvent<string>) =>
      dispatch(message.data);
    socket.onclose = () => {
      connected.value = false;
      socket = null;
      if (announce) {
        announce = false;
        useToastStore().createWarnToast(
          "Disconnected",
          "Real-time event updates disabled.",
        );
      }
      scheduleReconnect();
    };
  };

  const disconnect = (notify = false): void => {
    announce = false;

    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (socket) {
      // detach first: this socket's close event lands after a later connect() may have replaced it
      socket.onopen = null;
      socket.onmessage = null;
      socket.onclose = null;
      socket.close();
      socket = null;
    }
    connected.value = false;
    attempts.value = 0;

    if (notify) {
      useToastStore().createWarnToast(
        "Disconnected",
        "Real-time event updates disabled.",
      );
    }
  };

  // Detaches the dead socket first, so the operator's retry is not swallowed by
  // `if (socket) return` after the attempt cap stopped the backoff loop.
  const reconnect = (): void => {
    disconnect();
    announce = true;
    connect();
  };

  return {
    connected,
    attempts,
    endpoint,
    on,
    connect,
    disconnect,
    reconnect,
  };
});
