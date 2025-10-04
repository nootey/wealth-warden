<script setup lang="ts">
import {useSharedStore} from "../../services/stores/shared_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {computed, onMounted, provide, ref} from "vue";
import {useUserStore} from "../../services/stores/user_store.ts";
import type {Role, User} from "../../models/user_models.ts";
import filterHelper from "../../utils/filter_helper.ts";
import type {Column} from "../../services/filter_registry.ts";
import type {FilterObj} from "../../models/shared_models.ts";
import FilterMenu from "../components/filters/FilterMenu.vue";
import ActiveFilters from "../components/filters/ActiveFilters.vue";
import ActionRow from "../components/layout/ActionRow.vue";
import dateHelper from "../../utils/date_helper.ts";
import LoadingSpinner from "../components/base/LoadingSpinner.vue";
import ColumnHeader from "../components/base/ColumnHeader.vue";
import CustomPaginator from "../components/base/CustomPaginator.vue";
import UserForm from "../components/forms/UserForm.vue";
import InvitationsPaginated from "../components/data/InvitationsPaginated.vue";
import {useRouter} from "vue-router";
import {usePermissions} from "../../utils/use_permissions.ts";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const userStore = useUserStore();
const { hasPermission } = usePermissions();

onMounted(async () => {
    await userStore.getRoles();
})

const router = useRouter();
const apiPrefix = userStore.apiPrefix;

const createModal = ref(false);
const updateModal = ref(false);
const updateUserID = ref(null);

const loading = ref(false);
const records = ref<User[]>([]);
const roles = computed<Role[]>(() => userStore.roles);

const params = computed(() => {
    return {
        rowsPerPage: paginator.value.rowsPerPage,
        sort: sort.value,
        filters: filters.value,
    }
});

const rows = ref([10, 25, 50, 100]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
    total: 0,
    from: 0,
    to: 0,
    rowsPerPage: default_rows.value
});
const page = ref(1);
const sort = ref(filterHelper.initSort());
const filterStorageIndex = ref(apiPrefix+"-filters");
const filters = ref(JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"));
const filterOverlayRef = ref<any>(null);

const activeColumns = computed<Column[]>(() => [
    { field: 'display_name', header: 'Name', type: 'text'},
    { field: 'email', header: 'Email', type: 'text'},
    { field: 'role', header: 'Role', type: 'enum', options: roles.value, optionLabel: 'name'},
    { field: 'email_confirmed', header: 'Date', type: "date" },
]);

const invRef = ref<InstanceType<typeof InvitationsPaginated> | null>(null);

onMounted(async () => {
    await init();
});

async function init() {
    await getData();
}

async function getData(new_page: number|null = null) {

    loading.value = true;
    if(new_page)
        page.value = new_page;

    try {

        let payload = {
            ...params.value,
        };

        let paginationResponse = await sharedStore.getRecordsPaginated(
            apiPrefix,
            payload,
            page.value
        );

        records.value = paginationResponse.data;
        paginator.value.total = paginationResponse.total_records;
        paginator.value.to = paginationResponse.to;
        paginator.value.from = paginationResponse.from;
        loading.value = false;
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

function manipulateDialog(modal: string, value: any) {
    switch (modal) {
        case 'inviteUser': {
            createModal.value = value;
            break;
        }
        case 'updateUser': {
            updateModal.value = true;
            updateUserID.value = value;
            break;
        }
        default: {
            break;
        }
    }
}

async function handleEmit(emitType: any) {
    switch (emitType) {
        case 'completeOperation': {
            createModal.value = false;
            updateModal.value = false;
            await getData();
            invRef.value?.refresh();
            break;
        }
        case 'deleteUser': {
            createModal.value = false;
            updateModal.value = false;
            await getData();
            break;
        }
        default: {
            break;
        }
    }
}

async function onPage(event: any) {
    paginator.value.rowsPerPage = event.rows;
    page.value = (event.page+1)
    await getData();
}

function applyFilters(list: FilterObj[]){
    filters.value = filterHelper.mergeFilters(filters.value, list);
    localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
    getData();
    filterOverlayRef.value.hide();
}

function clearFilters(){
    filters.value = [];
    localStorage.removeItem(filterStorageIndex.value);
    cancelFilters();
    getData();
}

function cancelFilters(){
    filterOverlayRef.value.hide();
}

function removeFilter(index: number) {
    if (index < 0 || index >= filters.value.length) return;

    const next = filters.value.slice();
    next.splice(index, 1);
    filters.value = next;

    if (filters.value.length > 0) {
        localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
    } else {
        localStorage.removeItem(filterStorageIndex.value);
    }

    getData();
}

function switchSort(column:string) {
    if (sort.value.field === column) {
        sort.value.order = filterHelper.toggleSort(sort.value.order);
    } else {
        sort.value.order = 1;
    }
    sort.value.field = column;
    getData();
}

function toggleFilterOverlay(event: any) {
    filterOverlayRef.value.toggle(event);
}

provide("switchSort", switchSort);
provide("removeFilter", removeFilter);

</script>

<template>

    <Popover ref="filterOverlayRef" class="rounded-popover" :style="{width: '420px'}" :breakpoints="{'775px': '90vw'}">
        <FilterMenu
                v-model:value="filters"
                :columns="activeColumns"
                :apiSource="apiPrefix"
                @apply="(list) => applyFilters(list)"
                @clear="clearFilters"
                @cancel="cancelFilters"
        />
    </Popover>

    <Dialog class="rounded-dialog" v-model:visible="createModal" :breakpoints="{'501px': '90vw'}"
            :modal="true" :style="{width: '500px'}" header="Invite user">
        <UserForm mode="create" :roles="roles"
                  @completeOperation="handleEmit('completeOperation')">
        </UserForm>
    </Dialog>

    <Dialog position="right" class="rounded-dialog" v-model:visible="updateModal" :breakpoints="{'501px': '90vw'}"
            :modal="true" :style="{width: '500px'}" header="User details">
        <UserForm mode="update" :roles="roles" :recordId="updateUserID"
                  @completeOperation="handleEmit('completeOperation')"
                  @completeUserDelete="handleEmit('deleteUser')">
        </UserForm>
    </Dialog>

    <main class="flex flex-column w-full p-2 align-items-center" style="height: 100vh;">
        <div class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
             style="border: 1px solid var(--border-color); background: var(--background-secondary); max-width: 1000px;">

            <div class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full">
                <div style="font-weight: bold;">Users</div>
                <i v-if="hasPermission('manage_roles')" class="pi pi-external-link hover-icon mr-auto text-sm" @click="router.push('settings/roles')" v-tooltip="'Go to roles settings.'"></i>
                <Button class="main-button"
                        @click="manipulateDialog('inviteUser', true)">
                    <div class="flex flex-row gap-1 align-items-center">
                        <i class="pi pi-plus"></i>
                        <span> New </span>
                        <span class="mobile-hide"> User </span>
                    </div>
                </Button>
            </div>

            <div class="flex flex-row justify-content-between align-items-center p-1 gap-3 w-full border-round-md"
                 style="border: 1px solid var(--border-color);background: var(--background-secondary);">

                <ActionRow>
                    <template #activeFilters>
                        <ActiveFilters :activeFilters="filters" :showOnlyActive="false" activeFilter="" />
                    </template>
                    <template #filterButton>
                        <div class="hover-icon flex flex-row align-items-center gap-2" @click="toggleFilterOverlay($event)"
                             style="padding: 0.5rem 1rem; border-radius: 8px; border: 1px solid var(--border-color)">
                            <i class="pi pi-filter" style="font-size: 0.845rem"></i>
                            <div>Filter</div>
                        </div>
                    </template>
                </ActionRow>
            </div>

            <div class="flex flex-row gap-2 w-full">
                <div class="w-full">
                    <DataTable class="w-full enhanced-table" dataKey="id" :loading="loading" :value="records"
                               :rowHover="true" :showGridlines="false">
                        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
                        <template #loading> <LoadingSpinner></LoadingSpinner> </template>
                        <template #footer>
                            <CustomPaginator :paginator="paginator" :rows="rows" @onPage="onPage"/>
                        </template>

                        <Column v-for="col of activeColumns" :key="col.field" :field="col.field" style="width: 25%">
                            <template #header >
                                <ColumnHeader  :header="col.header" :field="col.field" :sort="sort"></ColumnHeader>
                            </template>
                            <template #body="{ data, field }">
                                <template v-if="field === 'email_confirmed'">
                                    {{ dateHelper.formatDate(data?.email_confirmed, true) }}
                                </template>
                                <template v-else-if="field === 'display_name'">
                                    <span class="hover-icon font-bold" @click="manipulateDialog('updateUser', data.id)">
                                        {{ data[field] }}
                                    </span>
                                </template>
                                <template v-else-if="field === 'role'">
                                    <span>
                                        {{ data[field]["name"] }}
                                    </span>
                                </template>
                                <template v-else>
                                    {{ data[field] }}
                                </template>
                            </template>
                        </Column>
                    </DataTable>
                </div>
            </div>

            <label>Invitations</label>
            <div class="flex flex-row gap-2 w-full">
                <InvitationsPaginated ref="invRef"></InvitationsPaginated>
            </div>

        </div>
    </main>
</template>

<style scoped>

</style>