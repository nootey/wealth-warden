import { ref, watch } from "vue";
import { useAccountStore } from "../services/stores/account_store.ts";
import type { Account, AvailableBalance } from "../models/account_models.ts";

export function useAvailableBalance(account: () => Account | null | undefined) {
  const accountStore = useAccountStore();
  const available = ref<AvailableBalance | null>(null);

  watch(
    () => account()?.id,
    async (id) => {
      available.value = null;
      if (!id) return;
      const res = await accountStore.getAvailableBalance(id).catch(() => null);
      if (account()?.id === id) available.value = res;
    },
    { immediate: true },
  );

  return { available };
}
