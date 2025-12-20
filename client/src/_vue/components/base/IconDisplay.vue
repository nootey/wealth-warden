<script setup lang="ts">
import { ref } from "vue";

defineProps<{
  event?: string;
}>();

const properties = ref<any>({
  event: {
    login: { icon: "pi pi-unlock" },
    register: { icon: "pi pi-user-plus" },
    create: { icon: "pi pi-plus" },
    update: { icon: "pi pi-pencil" },
    delete: { icon: "pi pi-trash" },
    "confirm-email": { icon: "pi pi-envelope" },
    "password-reset": { icon: "pi pi-undo" },
    resend: { icon: "pi pi-sync" },
  },
});

function getProperty(
  property: string,
  type: string,
  value: string,
): string | null {
  if (
    property &&
    type &&
    value &&
    Object.prototype.hasOwnProperty.call(properties.value, property) &&
    Object.prototype.hasOwnProperty.call(properties.value[property], type) &&
    Object.prototype.hasOwnProperty.call(
      properties.value[property][type],
      value,
    )
  ) {
    return properties.value[property][type][value];
  } else if (property && type) {
    return type;
  }
  return null;
}
</script>

<template>
  <div v-if="event" class="flex flex-row align-items-center">
    <i v-tooltip="getProperty('event', event, ``)" class="mobile-hide">
      <span
        class="flex flex-row align-items-center justify-items-center text-center custom-marker shadow-2"
        style="
          background: var(--background-primary);
          color: var(--text-primary);
        "
      >
        <span
          :class="getProperty('event', event, 'icon')"
          style="font-size: 0.875rem"
        />
      </span>
    </i>
    <div class="event-text">
      {{ event }}
    </div>
  </div>
  <div v-else>
    {{ "none" }}
  </div>
</template>

<style scoped lang="scss">
.custom-marker {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 1px solid var(--border-color);
  background: var(--background-primary);
  color: var(--text-primary);

  span {
    font-size: 0.875rem;
    line-height: 1;
  }
}

.event-text {
  margin-left: 0.5rem;
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  text-transform: uppercase;
  font-weight: 700;
  font-size: 13px;
  letter-spacing: 0.3px;
  color: var(--accent-primary);
}

.dot {
  height: 8px;
  width: 8px;
  border-radius: 50%;
  display: inline-block;
  margin-top: 9px;
  margin-left: -2px;
}
</style>
