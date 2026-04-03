import { describe, it, expect, vi, beforeEach } from "vitest";
import { ApiError } from "./api_models.ts";

vi.mock("../stores/auth_store.ts", () => ({
  useAuthStore: vi.fn(() => ({ isAuthenticated: false, logoutUser: vi.fn() })),
}));

vi.stubGlobal("window", { location: { origin: "http://localhost" } });

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

// import after stubs are set up
const { default: apiClient } = await import("./api_client.ts");

function mockResponse(status: number, body: unknown = null): Response {
  const bodyStr = body != null ? JSON.stringify(body) : "";
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(bodyStr),
    blob: () => Promise.resolve(new Blob([bodyStr])),
  } as Response;
}

describe("apiClient", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  describe("param serialization", () => {
    it("serializes primitive params", async () => {
      mockFetch.mockResolvedValue(mockResponse(200, {}));
      await apiClient.get("test", { params: { page: 1, active: true } });
      const url = new URL(mockFetch.mock.calls[0]![0], "http://localhost");
      expect(url.searchParams.get("page")).toBe("1");
      expect(url.searchParams.get("active")).toBe("true");
    });

    it("serializes nested objects with bracket notation", async () => {
      mockFetch.mockResolvedValue(mockResponse(200, {}));
      await apiClient.get("test", {
        params: { sort: { field: "name", order: -1 } },
      });
      const url = new URL(mockFetch.mock.calls[0]![0], "http://localhost");
      expect(url.searchParams.get("sort[field]")).toBe("name");
      expect(url.searchParams.get("sort[order]")).toBe("-1");
    });

    it("serializes arrays with indexed bracket notation", async () => {
      mockFetch.mockResolvedValue(mockResponse(200, {}));
      await apiClient.get("test", { params: { ids: [1, 2, 3] } });
      const url = new URL(mockFetch.mock.calls[0]![0], "http://localhost");
      expect(url.searchParams.get("ids[0]")).toBe("1");
      expect(url.searchParams.get("ids[1]")).toBe("2");
      expect(url.searchParams.get("ids[2]")).toBe("3");
    });

    it("skips null and undefined params", async () => {
      mockFetch.mockResolvedValue(mockResponse(200, {}));
      await apiClient.get("test", {
        params: { a: null, b: undefined, c: "keep" },
      });
      const url = new URL(mockFetch.mock.calls[0]![0], "http://localhost");
      expect(url.searchParams.has("a")).toBe(false);
      expect(url.searchParams.has("b")).toBe(false);
      expect(url.searchParams.get("c")).toBe("keep");
    });
  });

  describe("error handling", () => {
    it("throws ApiError on non-ok response", async () => {
      mockFetch.mockResolvedValue(
        mockResponse(500, { message: "Server error" }),
      );
      await expect(apiClient.get("test")).rejects.toBeInstanceOf(ApiError);
    });

    it("includes server message and status in ApiError", async () => {
      mockFetch.mockResolvedValue(mockResponse(400, { message: "Bad input" }));
      await expect(apiClient.get("test")).rejects.toMatchObject({
        message: "Bad input",
        status: 400,
      });
    });

    it("sets isNetworkError when fetch throws", async () => {
      mockFetch.mockRejectedValue(new Error("Failed to fetch"));
      await expect(apiClient.get("test")).rejects.toMatchObject({
        isNetworkError: true,
        status: 0,
      });
    });
  });

  describe("request config", () => {
    it("sends JSON body with correct Content-Type", async () => {
      mockFetch.mockResolvedValue(mockResponse(200, {}));
      await apiClient.post("test", { name: "foo" });
      const [, init] = mockFetch.mock.calls[0]!;
      expect((init as RequestInit).headers).toMatchObject({
        "Content-Type": "application/json",
      });
      expect((init as RequestInit).body).toBe(JSON.stringify({ name: "foo" }));
    });

    it("does not set Content-Type for FormData", async () => {
      mockFetch.mockResolvedValue(mockResponse(200, {}));
      await apiClient.post("test", new FormData());
      const [, init] = mockFetch.mock.calls[0]!;
      expect(
        (init as RequestInit & { headers: Record<string, string> }).headers[
          "Content-Type"
        ],
      ).toBeUndefined();
    });

    it("includes credentials", async () => {
      mockFetch.mockResolvedValue(mockResponse(200, {}));
      await apiClient.get("test");
      const [, init] = mockFetch.mock.calls[0]!;
      expect((init as RequestInit).credentials).toBe("include");
    });
  });
});
