<script setup lang="ts">
import {ref} from "vue";

defineProps(['event']);

const properties = ref<any>({
  event: {
    'login': { icon: 'pi pi-unlock' },
    'register': { icon: 'pi pi-user-plus' },
    'create': { icon: 'pi pi-plus' },
    'update': { icon: 'pi pi-pencil' },
    'delete': { icon: 'pi pi-trash' },
    'email-validate': { icon: 'pi pi-envelope' },
    'password-reset': { icon: 'pi pi-undo' },
    'return': { icon: 'pi pi-backward' },
    'decode-jwt': { icon: 'pi pi-chart-pie' },
    'validate-jwt': { icon: 'pi pi-id-card' },
    'validate-licence': { icon: 'pi pi-id-card' },
    'claim': { icon: 'pi pi-send' },
    'restore': { icon: 'pi pi-sync' },
  },
});

function getProperty(property: string, type: any, value: any): string|null {
  if (property && type && value && properties.value.hasOwnProperty(property) &&
      properties.value[property].hasOwnProperty(type) &&
      properties.value[property][type].hasOwnProperty(value)) {
    return properties.value[property][type][value];
  } else if (property && type) {
    return type;
  }
  return null;
}
</script>

<template>
  <div v-if="event" class="flex flex-row align-items-center">
    <i v-tooltip="getProperty('event', event, ``)">
            <span class="flex flex-row align-items-center justify-items-center text-center custom-marker shadow-1"
                  style="background: var(--background-primary); color: var(--text-primary);">
                <span :class="getProperty('event', event, 'icon')"></span>
            </span>
    </i>
    <div  class="event-text">
      {{ event }}
    </div>
  </div>
  <div v-else> {{ "none"}} </div>
</template>

<style scoped lang="scss">
.custom-marker {
  display: flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  border: 1px solid var(--border-color);
  color: var(--background-primary);
  padding: 7px;
}
.event-text {
  margin-right: 45px;
  border-radius: 8px;
  padding: 0.5em 0.5rem;
  text-transform: uppercase;
  font-weight: 700;
  font-size: 13px;
  letter-spacing: .3px;
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