import { defineStore } from "pinia";
import { useToast } from "primevue/usetoast";

export const useToastStore = defineStore("toast", () => {
  const toast = useToast();
  const isMobile = window.matchMedia("(max-width: 768px)").matches;
  const group = isMobile ? "bc" : "br";

  const errorResponseToast = (error: unknown) => {
    console.error("triggered error", error);

    const isAxiosError = (
      err: unknown,
    ): err is {
      response?: { data?: { title?: string; message?: string } };
      code?: string;
      message?: string;
    } => {
      return typeof err === "object" && err !== null;
    };

    let summary = "Unexpected Error";
    let detail = "An unknown error occurred.";

    if (isAxiosError(error)) {
      const data = error.response?.data;

      if (data?.title || data?.message) {
        summary = data.title ?? "Error";
        detail = data.message ?? "Something went wrong.";
      }

      if (error.code === "ERR_NETWORK" || error.message === "Network Error") {
        summary = "Server unreachable";
        detail = "The server is currently not reachable.";
      } else if (error.message) {
        detail = error.message;
      }
    }

    toast.add({
      severity: "error",
      summary,
      detail,
      life: isMobile ? 2500 : 5000,
      group,
    });
  };

  type ToastResponse = {
    data?: {
      title?: string;
      message?: string;
    };
    title?: string;
    message?: string;
  };

  const successResponseToast = (response: ToastResponse) => {
    const data = response?.data || response;
    if (data?.title || data?.message) {
      toast.add({
        severity: "success",
        summary: data.title ?? "Success",
        detail: data.message ?? "",
        life: isMobile ? 1500 : 3000,
        group,
      });
    }
  };

  const infoResponseToast = (response: ToastResponse) => {
    const data = response?.data || response;
    if (data?.title || data?.message) {
      toast.add({
        severity: "info",
        summary: data.title ?? "Info",
        detail: data.message ?? "",
        life: isMobile ? 1000 : 2000,
        group,
      });
    }
  };

  const createInfoToast = (title: string, msg: string) => {
    if (title && msg) {
      toast.add({
        severity: "info",
        summary: title,
        detail: msg,
        life: isMobile ? 1000 : 2000,
        group,
      });
    }
  };

  return {
    errorResponseToast,
    successResponseToast,
    infoResponseToast,
    createInfoToast,
  };
});
