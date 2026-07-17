import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useWsStore } from "./ws_store.ts";

const authState = vi.hoisted(() => ({
  isAuthenticated: true,
  logout: vi.fn(),
}));
vi.mock("./auth_store.ts", () => ({
  useAuthStore: () => authState,
}));

const toasts = vi.hoisted(() => ({
  createInfoToast: vi.fn(),
  createWarnToast: vi.fn(),
}));
vi.mock("./toast_store.ts", () => ({
  useToastStore: () => toasts,
}));

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  url: string;
  closed = false;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: ((event: { code: number }) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  close(): void {
    this.closed = true;
  }

  // a real socket only notifies through handlers that are still attached
  fireOpen(): void {
    this.onopen?.();
  }

  fireMessage(data: string): void {
    this.onmessage?.({ data });
  }

  fireClose(code = 1006): void {
    this.onclose?.({ code });
  }
}

const setTimeoutSpy = vi.fn(
  (handler: () => void, ms?: number) =>
    globalThis.setTimeout(handler, ms) as unknown as number,
);
const clearTimeoutSpy = vi.fn((id: number) => {
  globalThis.clearTimeout(id);
});

const location = { protocol: "http:", host: "localhost:5000" };

// the socket ws_store is currently holding
const socket = (): FakeWebSocket => {
  const latest = FakeWebSocket.instances.at(-1);
  if (!latest) throw new Error("no socket was created");
  return latest;
};

const lastDelay = (): number | undefined =>
  setTimeoutSpy.mock.calls.at(-1)?.[1];

const encode = (type: string, payload?: unknown): string =>
  JSON.stringify({ type, payload });

describe("wsStore", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    setActivePinia(createPinia());

    FakeWebSocket.instances = [];
    setTimeoutSpy.mockClear();
    clearTimeoutSpy.mockClear();
    toasts.createInfoToast.mockClear();
    toasts.createWarnToast.mockClear();

    authState.isAuthenticated = true;
    authState.logout.mockClear();
    location.protocol = "http:";
    location.host = "localhost:5000";

    // pins the jitter factor to 0.5, making every backoff deterministic
    vi.spyOn(Math, "random").mockReturnValue(0);

    vi.stubGlobal("WebSocket", FakeWebSocket);
    vi.stubGlobal("window", {
      location,
      setTimeout: setTimeoutSpy,
      clearTimeout: clearTimeoutSpy,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  describe("connect", () => {
    it("opens an insecure socket from an insecure page", () => {
      useWsStore().connect();
      expect(socket().url).toBe("ws://localhost:5000/api/ws");
    });

    it("opens a secure socket from a secure page", () => {
      location.protocol = "https:";
      useWsStore().connect();
      expect(socket().url).toBe("wss://localhost:5000/api/ws");
    });

    it("ignores a second connect while a socket is already held", () => {
      const store = useWsStore();
      store.connect();
      store.connect();
      expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it("reports connected only once the socket opens", () => {
      const store = useWsStore();
      store.connect();
      expect(store.connected).toBe(false);

      socket().fireOpen();
      expect(store.connected).toBe(true);
    });
  });

  describe("dispatch", () => {
    it("delivers the payload to handlers of that event type", () => {
      const store = useWsStore();
      const handler = vi.fn();
      store.on("report.completed", handler);
      store.connect();

      socket().fireMessage(encode("report.completed", { report_id: 7 }));

      expect(handler).toHaveBeenCalledTimes(1);
      expect(handler).toHaveBeenCalledWith({ report_id: 7 });
    });

    it("delivers to every handler registered for the type", () => {
      const store = useWsStore();
      const first = vi.fn();
      const second = vi.fn();
      store.on("notification.created", first);
      store.on("notification.created", second);
      store.connect();

      socket().fireMessage(encode("notification.created"));

      expect(first).toHaveBeenCalledTimes(1);
      expect(second).toHaveBeenCalledTimes(1);
    });

    it("swallows malformed json rather than throwing", () => {
      const store = useWsStore();
      const handler = vi.fn();
      store.on("report.completed", handler);
      store.connect();

      expect(() => socket().fireMessage("not json")).not.toThrow();
      expect(handler).not.toHaveBeenCalled();
    });

    it("ignores an event type nobody is listening for", () => {
      const store = useWsStore();
      const handler = vi.fn();
      store.on("report.completed", handler);
      store.connect();

      expect(() => socket().fireMessage(encode("report.failed"))).not.toThrow();
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe("on", () => {
    it("stops delivering once unsubscribed", () => {
      const store = useWsStore();
      const handler = vi.fn();
      const off = store.on("notification.created", handler);
      store.connect();
      off();

      socket().fireMessage(encode("notification.created"));

      expect(handler).not.toHaveBeenCalled();
    });

    it("unsubscribing one handler leaves its siblings registered", () => {
      const store = useWsStore();
      const removed = vi.fn();
      const kept = vi.fn();
      const off = store.on("notification.created", removed);
      store.on("notification.created", kept);
      store.connect();
      off();

      socket().fireMessage(encode("notification.created"));

      expect(removed).not.toHaveBeenCalled();
      expect(kept).toHaveBeenCalledTimes(1);
    });
  });

  describe("reconnect", () => {
    it("opens a fresh socket after an unintentional close", () => {
      const store = useWsStore();
      store.connect();

      socket().fireClose();
      expect(store.connected).toBe(false);

      vi.advanceTimersByTime(500);
      expect(FakeWebSocket.instances).toHaveLength(2);
    });

    it("logs out instead of reconnecting when the server revokes the session", () => {
      const store = useWsStore();
      store.connect();
      socket().fireOpen();

      socket().fireClose(4001);

      expect(store.connected).toBe(false);
      expect(toasts.createWarnToast).toHaveBeenCalledWith(
        "Logged out",
        "This session was revoked.",
      );
      expect(authState.logout).toHaveBeenCalledTimes(1);
      expect(setTimeoutSpy).not.toHaveBeenCalled();
      expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it("backs off exponentially and caps at MAX_BACKOFF_MS", () => {
      const store = useWsStore();
      store.connect();

      const delays: (number | undefined)[] = [];
      for (let i = 0; i < 7; i++) {
        socket().fireClose();
        delays.push(lastDelay());
        vi.advanceTimersByTime(30000);
      }

      expect(delays).toEqual([500, 1000, 2000, 4000, 8000, 15000, 15000]);
    });

    it("gives up after MAX_ATTEMPTS so an expired session cannot loop forever", () => {
      const store = useWsStore();
      store.connect();

      for (let i = 0; i < 10; i++) {
        socket().fireClose();
        vi.advanceTimersByTime(30000);
      }
      expect(setTimeoutSpy).toHaveBeenCalledTimes(10);
      expect(FakeWebSocket.instances).toHaveLength(11);

      socket().fireClose();

      expect(setTimeoutSpy).toHaveBeenCalledTimes(10);
      expect(vi.getTimerCount()).toBe(0);
    });

    it("resets the backoff once a socket opens successfully", () => {
      const store = useWsStore();
      store.connect();

      socket().fireClose();
      expect(lastDelay()).toBe(500);
      vi.advanceTimersByTime(500);

      socket().fireClose();
      expect(lastDelay()).toBe(1000);
      vi.advanceTimersByTime(1000);

      socket().fireOpen();
      socket().fireClose();
      expect(lastDelay()).toBe(500);
    });

    it("does not reconnect once the user is unauthenticated", () => {
      authState.isAuthenticated = false;
      const store = useWsStore();
      store.connect();

      socket().fireClose();

      expect(setTimeoutSpy).not.toHaveBeenCalled();
      expect(vi.getTimerCount()).toBe(0);
    });
  });

  describe("disconnect", () => {
    it("closes the socket and clears connected", () => {
      const store = useWsStore();
      store.connect();
      const opened = socket();
      opened.fireOpen();

      store.disconnect();

      expect(opened.closed).toBe(true);
      expect(store.connected).toBe(false);
    });

    it("does not reconnect on the close event it triggered", () => {
      const store = useWsStore();
      store.connect();
      const opened = socket();

      store.disconnect();
      opened.fireClose();

      expect(setTimeoutSpy).not.toHaveBeenCalled();
      expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it("cancels a reconnect that is already pending", () => {
      const store = useWsStore();
      store.connect();
      socket().fireClose();
      expect(vi.getTimerCount()).toBe(1);

      store.disconnect();

      expect(clearTimeoutSpy).toHaveBeenCalledTimes(1);
      expect(vi.getTimerCount()).toBe(0);
      vi.advanceTimersByTime(30000);
      expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it("resets the backoff for the next connect", () => {
      const store = useWsStore();
      store.connect();
      socket().fireClose();
      vi.advanceTimersByTime(500);
      socket().fireClose();
      expect(lastDelay()).toBe(1000);

      store.disconnect();
      store.connect();
      socket().fireClose();

      expect(lastDelay()).toBe(500);
    });

    it("stops dispatching messages that arrive on a socket it closed", () => {
      const store = useWsStore();
      const handler = vi.fn();
      store.on("notification.created", handler);
      store.connect();
      const opened = socket();

      store.disconnect();
      opened.fireMessage(encode("notification.created"));

      expect(handler).not.toHaveBeenCalled();
    });

    it("stays disconnected when a socket it closed opens late", () => {
      const store = useWsStore();
      store.connect();
      const opened = socket();

      store.disconnect();
      opened.fireOpen();

      expect(store.connected).toBe(false);
    });

    it("stays silent through the login connect and the backoff loop", () => {
      const store = useWsStore();
      store.connect();

      socket().fireOpen();
      socket().fireClose();
      vi.advanceTimersByTime(500);
      socket().fireOpen();

      expect(toasts.createInfoToast).not.toHaveBeenCalled();
      expect(toasts.createWarnToast).not.toHaveBeenCalled();
    });

    it("announces a hand-driven connect once it opens", () => {
      const store = useWsStore();
      store.reconnect();

      expect(toasts.createInfoToast).not.toHaveBeenCalled();
      socket().fireOpen();
      expect(toasts.createInfoToast).toHaveBeenCalledTimes(1);
    });

    it("announces a hand-driven connect that fails to open", () => {
      const store = useWsStore();
      store.reconnect();

      socket().fireClose();
      expect(toasts.createWarnToast).toHaveBeenCalledTimes(1);

      // the silent backoff owns every retry from here on
      vi.advanceTimersByTime(500);
      socket().fireOpen();
      expect(toasts.createInfoToast).not.toHaveBeenCalled();
    });

    it("announces a hand-driven disconnect", () => {
      const store = useWsStore();
      store.connect();
      socket().fireOpen();

      store.disconnect(true);

      expect(toasts.createWarnToast).toHaveBeenCalledTimes(1);
    });

    it("stays silent when the user logs out", () => {
      const store = useWsStore();
      store.connect();
      socket().fireOpen();

      store.disconnect();

      expect(toasts.createWarnToast).not.toHaveBeenCalled();
    });

    it("reconnects on demand after the attempt cap gave up", () => {
      const store = useWsStore();
      store.connect();

      for (let i = 0; i < 10; i++) {
        socket().fireClose();
        vi.advanceTimersByTime(30000);
      }
      socket().fireClose();
      expect(vi.getTimerCount()).toBe(0);

      const before = FakeWebSocket.instances.length;
      store.reconnect();

      expect(FakeWebSocket.instances).toHaveLength(before + 1);
      expect(store.attempts).toBe(0);
    });

    it("connects again after a manual disconnect", () => {
      const store = useWsStore();
      store.connect();
      socket().fireOpen();

      store.disconnect();
      expect(store.connected).toBe(false);

      store.reconnect();
      socket().fireOpen();

      expect(FakeWebSocket.instances).toHaveLength(2);
      expect(store.connected).toBe(true);
    });

    it("replaces a socket still mid-handshake rather than no-opping", () => {
      const store = useWsStore();
      store.connect();
      const inFlight = socket();

      // never opened: `connected` is false while `socket` is still held
      store.reconnect();

      expect(inFlight.closed).toBe(true);
      expect(FakeWebSocket.instances).toHaveLength(2);
    });

    it("ignores a late close from a socket it has already replaced", () => {
      const store = useWsStore();
      store.connect();
      const stale = socket();

      store.disconnect();
      store.connect();
      const live = socket();
      live.fireOpen();

      // the close for the socket disconnect() closed only reaches us now
      stale.fireClose();

      expect(store.connected).toBe(true);
      expect(setTimeoutSpy).not.toHaveBeenCalled();

      vi.advanceTimersByTime(30000);
      expect(FakeWebSocket.instances).toHaveLength(2);
    });
  });
});
