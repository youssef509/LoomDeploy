<script setup lang="ts">
import type { Project, GitHubAppStatus, GitHubRepo } from '~/types'
import { z } from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

definePageMeta({ middleware: 'auth' })

const api = useApi()
const toast = useToast()
const route = useRoute()
const router = useRouter()

const { data: projects, pending, refresh } = useAsyncData('projects', () => api.getProjects(), { default: () => [] })
const { data: ghStatus } = useAsyncData('gh-status-proj', () => api.getGitHubAppStatus(), {
  default: () => ({ configured: false, connected: false }) as GitHubAppStatus
})

const isNewModalOpen = ref(!!route.query.new)
const deleteConfirmOpen = ref(false)
const projectToDelete = ref<Project | null>(null)

// ── Track / step state ──────────────────────────────────────────────────────
const track = ref<null | 'app' | 'service'>(null)
const serviceStep = ref<'grid' | 'form'>('grid')

// ── Service templates ────────────────────────────────────────────────────────
interface EnvVarField { key: string; label: string; placeholder: string; required: boolean; secret: boolean; default?: string }
interface ServiceTemplate { id: string; name: string; description: string; icon: string; category: string; docker_image: string; versions: string[]; default_port: number; volume_mount?: string; env_var_fields: EnvVarField[] }

const { data: templates } = await useAsyncData<ServiceTemplate[]>(
  'service-templates',
  () => $fetch('/api/services/templates'),
  { default: () => [] as ServiceTemplate[] }
)
const selectedTemplate = ref<ServiceTemplate | null>(null)
const serviceImage = ref('')
const serviceVolumeEnabled = ref(true)
const serviceName = ref('')
const serviceEnvVars = ref<{ key: string; value: string; secret: boolean }[]>([])
const serviceCreating = ref(false)

function selectTemplate(t: ServiceTemplate) {
  selectedTemplate.value = t
  serviceImage.value = t.docker_image
  serviceName.value = t.name.toLowerCase().replace(/\s+/g, '-')
  serviceEnvVars.value = t.env_var_fields.map(f => ({ key: f.key, value: f.default ?? '', secret: f.secret }))
  serviceStep.value = 'form'
}

async function submitService() {
  if (!selectedTemplate.value) return
  serviceCreating.value = true
  try {
    const auth = useAuthStore()
    const res = await $fetch<{ project: Project }>('/api/services', {
      method: 'POST',
      headers: { Authorization: `Bearer ${auth.token}` },
      body: {
        template_id: selectedTemplate.value.id,
        name: serviceName.value,
        image: serviceImage.value,
        volume_enabled: serviceVolumeEnabled.value,
        env_vars: serviceEnvVars.value.filter(e => e.key && e.value).map(e => ({ key: e.key, value: e.value }))
      }
    })
    toast.add({ title: `${selectedTemplate.value.name} deployed!`, color: 'success' })
    isNewModalOpen.value = false
    refresh()
    router.push(`/projects/${res.project.id}`)
  } catch (err: any) {
    toast.add({ title: 'Deploy failed', description: err?.data?.message, color: 'error' })
  } finally {
    serviceCreating.value = false
  }
}

const templateImages: Record<string, string> = {
  postgres: '/one-click-services/postgresql.png',
  mysql: '/one-click-services/mysql.png',
  mariadb: '/one-click-services/mariadb.png',
  redis: '/one-click-services/redis-white-1.webp',
  mongodb: '/one-click-services/mongodb.png',
  wordpress: '/one-click-services/wordpress.png',
  ghost: '/one-click-services/ghost.png',
  minio: '/one-click-services/MinIO.png'
}

// ── App (git) state ──────────────────────────────────────────────────────────
const ghRepos = ref<GitHubRepo[]>([])
const ghReposLoading = ref(false)
const sourceMode = ref<'github' | 'git_url' | 'upload'>('github')
const ghConnected = computed(() => (ghStatus.value as GitHubAppStatus).phase === 'connected')

async function loadGhRepos() {
  ghReposLoading.value = true
  try { ghRepos.value = await api.getGitHubRepos() }
  catch { ghRepos.value = [] }
  finally { ghReposLoading.value = false }
}

function resetNewProject() {
  newProject.name = ''
  newProject.source_type = 'git'
  newProject.repository_url = ''
  newProject.branch = 'main'
  newProject.container_port = 3000
}

watch(sourceMode, (mode) => {
  newProject.source_type = mode === 'upload' ? 'upload' : 'git'
})

watch(isNewModalOpen, (open) => {
  if (open) {
    track.value = null
    serviceStep.value = 'grid'
    selectedTemplate.value = null
    resetNewProject()
    sourceMode.value = ghConnected.value ? 'github' : 'git_url'
    if (ghConnected.value && ghRepos.value.length === 0) loadGhRepos()
  }
})

function selectRepo(repo: GitHubRepo) {
  newProject.repository_url = repo.clone_url
  newProject.branch = repo.default_branch
  if (!newProject.name) newProject.name = repo.full_name.split('/')[1] ?? ''
}

const search = ref('')
const filteredProjects = computed(() => {
  const q = search.value.toLowerCase().trim()
  if (!q) return projects.value as Project[]
  return (projects.value as Project[]).filter(p =>
    p.name.toLowerCase().includes(q) || p.domain.toLowerCase().includes(q)
  )
})

const schema = z.object({
  name: z.string().min(1, 'Name is required').max(50),
  source_type: z.enum(['git', 'upload'] as const),
  repository_url: z.string().optional(),
  branch: z.string().optional(),
  container_port: z.coerce.number().min(1).max(65535).default(3000)
})

type Schema = z.output<typeof schema>

const newProject = reactive<Partial<Schema>>({
  name: '',
  source_type: 'git',
  repository_url: '',
  branch: 'main',
  container_port: 3000
})

const creating = ref(false)

async function onCreateProject(event: FormSubmitEvent<Schema>) {
  creating.value = true
  try {
    const created = await api.createProject(event.data)
    toast.add({ title: 'Project created', color: 'success' })
    isNewModalOpen.value = false
    refresh()
    router.push(`/projects/${created.id}`)
  }
  catch (err: any) {
    toast.add({ title: 'Failed to create project', description: err?.data?.message, color: 'error' })
  }
  finally {
    creating.value = false
  }
}

function confirmDelete(project: Project) {
  projectToDelete.value = project
  deleteConfirmOpen.value = true
}

const deletingId = ref<string | null>(null)
async function executeDelete() {
  if (!projectToDelete.value) return
  deletingId.value = projectToDelete.value.id
  deleteConfirmOpen.value = false
  try {
    await api.deleteProject(projectToDelete.value.id)
    toast.add({ title: `"${projectToDelete.value.name}" deleted`, color: 'success' })
    refresh()
  }
  catch (err: any) {
    toast.add({ title: 'Delete failed', description: err?.data?.message, color: 'error' })
  }
  finally {
    deletingId.value = null
    projectToDelete.value = null
  }
}

const statusColor = (status?: string) => {
  if (status === 'running') return 'success'
  if (status === 'failed') return 'error'
  if (status === 'building') return 'warning'
  if (status === 'stopped') return 'neutral'
  return 'neutral'
}

const statusDot = (status?: string) => {
  if (status === 'running') return 'bg-green-500'
  if (status === 'failed') return 'bg-red-500'
  if (status === 'building') return 'bg-yellow-500 animate-pulse'
  if (status === 'stopped') return 'bg-zinc-500'
  return 'bg-zinc-600'
}

function deployProject(project: Project) {
  router.push(`/projects/${project.id}`)
}

const statusTextClass = (status?: string) => {
  if (status === 'running') return 'text-green-500'
  if (status === 'failed') return 'text-red-500'
  if (status === 'building') return 'text-amber-500'
  if (status === 'stopped') return 'text-zinc-500'
  return 'text-zinc-500'
}
</script>

<template>
  <UDashboardPanel id="projects">
    <template #header>
      <UDashboardNavbar title="Projects">
        <template #leading>
          <UDashboardSidebarCollapse class="!text-muted hover:!text-highlighted" />
        </template>
        <template #right>
          <UInput v-model="search" placeholder="Search…" icon="i-lucide-search" size="sm" class="w-44" />
          <UButton icon="i-lucide-plus" label="New Project" size="sm" @click="isNewModalOpen = true" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="p-6">
        <!-- Loading -->
        <div v-if="pending" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <div v-for="i in 3" :key="i" class="rounded-2xl border border-default bg-elevated/40 h-44 animate-pulse" />
        </div>

        <!-- Empty -->
        <div v-else-if="!(projects as Project[]).length" class="flex flex-col items-center justify-center py-28 gap-4">
          <div class="p-5 rounded-2xl bg-elevated">
            <UIcon name="i-lucide-box" class="size-10 text-muted" />
          </div>
          <div class="text-center">
            <h3 class="text-base font-semibold text-highlighted">No projects yet</h3>
            <p class="text-sm text-muted mt-1">Deploy your first application to get started.</p>
          </div>
          <UButton icon="i-lucide-plus" label="New Project" @click="isNewModalOpen = true" />
        </div>

        <!-- No search results -->
        <div v-else-if="!(projects as Project[]).length || !filteredProjects.length" class="flex flex-col items-center justify-center py-20 gap-3">
          <UIcon name="i-lucide-search-x" class="size-8 text-muted" />
          <p class="text-sm text-muted">No projects match <strong>"{{ search }}"</strong></p>
          <UButton variant="ghost" size="xs" @click="search = ''">Clear search</UButton>
        </div>

        <!-- Card grid -->
        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <div
            v-for="project in filteredProjects"
            :key="project.id"
            class="group relative flex flex-col rounded-2xl border border-default bg-elevated/50 hover:bg-elevated hover:border-primary/40 transition-all duration-200 overflow-hidden"
          >
            <!-- Top accent bar based on status -->
            <div
              class="h-0.5 w-full"
              :class="{
                'bg-green-500': project.latest_deployment?.status === 'running',
                'bg-red-500': project.latest_deployment?.status === 'failed',
                'bg-yellow-500': project.latest_deployment?.status === 'building',
                'bg-zinc-600': !project.latest_deployment || project.latest_deployment?.status === 'stopped'
              }"
            />

            <div class="p-5 flex flex-col gap-4 flex-1">
              <!-- Header row: icon + name + status dot -->
              <div class="flex items-start justify-between gap-2">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="shrink-0 p-2 rounded-xl bg-primary/10 border border-primary/20">
                    <UIcon name="i-lucide-box" class="size-5 text-primary" />
                  </div>
                  <div class="min-w-0">
                    <NuxtLink
                      :to="`/projects/${project.id}`"
                      class="block font-semibold text-highlighted truncate hover:text-primary transition-colors"
                    >
                      {{ project.name }}
                    </NuxtLink>
                    <a
                      :href="`https://${project.domain}`"
                      target="_blank"
                      class="text-xs font-mono text-muted hover:text-primary truncate flex items-center gap-0.5 mt-0.5 transition-colors"
                    >
                      {{ project.domain }}
                      <UIcon name="i-lucide-external-link" class="size-3 shrink-0" />
                    </a>
                  </div>
                </div>

                <!-- Status badge -->
                <div class="flex items-center gap-1.5 shrink-0">
                  <span class="size-2 rounded-full" :class="statusDot(project.latest_deployment?.status)" />
                  <span class="text-xs font-medium capitalize" :class="statusTextClass(project.latest_deployment?.status)">
                    {{ project.latest_deployment?.status ?? 'not deployed' }}
                  </span>
                </div>
              </div>

              <!-- Meta row -->
              <div class="flex items-center gap-3 text-xs text-muted">
                <span class="flex items-center gap-1">
                  <UIcon :name="project.source_type === 'git' ? 'i-simple-icons-git' : 'i-lucide-upload'" class="size-3.5" />
                  {{ project.source_type }}
                </span>
                <span v-if="project.branch" class="flex items-center gap-1">
                  <UIcon name="i-lucide-git-branch" class="size-3.5" />
                  {{ project.branch }}
                </span>
                <span class="flex items-center gap-1 ml-auto">
                  <UIcon name="i-lucide-calendar" class="size-3.5" />
                  {{ new Date(project.created_at).toLocaleDateString() }}
                </span>
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-2 mt-auto pt-3 border-t border-default">
                <UButton
                  :to="`/projects/${project.id}`"
                  size="xs"
                  variant="outline"
                  color="neutral"
                  icon="i-lucide-settings-2"
                  class="flex-1 justify-center"
                >
                  Manage
                </UButton>
                <UButton
                  size="xs"
                  variant="outline"
                  color="primary"
                  icon="i-lucide-rocket"
                  class="flex-1 justify-center"
                  @click="deployProject(project)"
                >
                  Deploy
                </UButton>
                <UButton
                  size="xs"
                  variant="ghost"
                  color="error"
                  icon="i-lucide-trash-2"
                  :loading="deletingId === project.id"
                  @click="confirmDelete(project)"
                />
              </div>
            </div>
          </div>

          <!-- New project card -->
          <button
            class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-default hover:border-primary/50 hover:bg-primary/5 transition-all duration-200 h-44 gap-3 text-muted hover:text-primary cursor-pointer"
            @click="isNewModalOpen = true"
          >
            <div class="p-3 rounded-xl bg-elevated">
              <UIcon name="i-lucide-plus" class="size-5" />
            </div>
            <span class="text-sm font-medium">New Project</span>
          </button>
        </div>
      </div>
    </template>
  </UDashboardPanel>

  <!-- Create project modal -->
  <UModal
    v-model:open="isNewModalOpen"
    :title="track === null ? 'New Project' : track === 'app' ? 'Deploy App' : (serviceStep === 'grid' ? 'Choose a Service' : (selectedTemplate?.name ?? 'Configure Service'))"
    :ui="{ body: 'space-y-0', content: track === 'service' && serviceStep === 'grid' ? 'max-w-2xl' : 'max-w-lg' }"
  >
    <template #body>

      <!-- ── Step 0: Track selector ── -->
      <div v-if="track === null" class="p-5 space-y-4">
        <p class="text-sm text-muted">What do you want to deploy?</p>
        <div class="grid grid-cols-2 gap-3">
          <button
            class="flex flex-col items-start gap-3 p-5 rounded-2xl border-2 border-default hover:border-primary/50 hover:bg-primary/5 transition-all text-left group"
            @click="track = 'app'"
          >
            <div class="p-2.5 rounded-xl bg-primary/10 border border-primary/20 group-hover:bg-primary/15">
              <UIcon name="i-lucide-git-branch" class="size-5 text-primary" />
            </div>
            <div>
              <p class="font-semibold text-sm text-highlighted">Deploy App</p>
              <p class="text-xs text-muted mt-0.5">From GitHub or any Git repository</p>
            </div>
          </button>
          <button
            class="flex flex-col items-start gap-3 p-5 rounded-2xl border-2 border-default hover:border-emerald-500/50 hover:bg-emerald-500/5 transition-all text-left group"
            @click="track = 'service'"
          >
            <div class="p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 group-hover:bg-emerald-500/15">
              <UIcon name="i-lucide-database" class="size-5 text-emerald-400" />
            </div>
            <div>
              <p class="font-semibold text-sm text-highlighted">Deploy Service</p>
              <p class="text-xs text-muted mt-0.5">One-click databases, caches & tools</p>
            </div>
          </button>
        </div>
        <div class="flex justify-end pt-1">
          <UButton variant="ghost" color="neutral" @click="isNewModalOpen = false">Cancel</UButton>
        </div>
      </div>

      <!-- ── Track: App (git) ── -->
      <UForm v-else-if="track === 'app'" :schema="schema" :state="newProject" class="p-4 space-y-4" @submit="onCreateProject">
        <div class="flex items-center gap-2 mb-1">
          <UButton size="xs" variant="ghost" color="neutral" icon="i-lucide-arrow-left" @click="track = null" />
          <span class="text-xs text-muted">Back</span>
        </div>

        <div class="grid grid-cols-3 gap-2">
          <button type="button" class="flex flex-col items-center gap-1.5 p-3 rounded-xl border-2 text-sm transition-all"
            :class="sourceMode === 'github' ? 'border-primary bg-primary/10' : 'border-default hover:border-primary/40'"
            @click="sourceMode = 'github'">
            <UIcon name="i-simple-icons-github" class="size-5" :class="sourceMode === 'github' ? 'text-primary' : 'text-muted'" />
            <span class="font-medium text-xs" :class="sourceMode === 'github' ? 'text-primary' : 'text-highlighted'">GitHub</span>
            <UBadge :color="ghConnected ? 'success' : 'neutral'" size="xs" variant="subtle">{{ ghConnected ? 'Connected' : 'Not connected' }}</UBadge>
          </button>
          <button type="button" class="flex flex-col items-center gap-1.5 p-3 rounded-xl border-2 text-sm transition-all"
            :class="sourceMode === 'git_url' ? 'border-primary bg-primary/10' : 'border-default hover:border-primary/40'"
            @click="sourceMode = 'git_url'">
            <UIcon name="i-lucide-link" class="size-5" :class="sourceMode === 'git_url' ? 'text-primary' : 'text-muted'" />
            <span class="font-medium text-xs" :class="sourceMode === 'git_url' ? 'text-primary' : 'text-highlighted'">Git URL</span>
            <span class="text-xs text-muted">GitLab, Gitea…</span>
          </button>
          <button type="button" disabled class="flex flex-col items-center gap-1.5 p-3 rounded-xl border-2 border-dashed border-default text-muted opacity-50 cursor-not-allowed">
            <UIcon name="i-lucide-upload" class="size-5" />
            <span class="font-medium text-xs">Upload</span>
            <UBadge color="neutral" size="xs" variant="subtle">Soon</UBadge>
          </button>
        </div>

        <template v-if="sourceMode === 'github'">
          <div v-if="ghConnected" class="space-y-1.5">
            <label class="text-sm font-medium text-highlighted">Repository</label>
            <div v-if="ghReposLoading" class="flex items-center gap-2 text-sm text-muted py-3 px-1">
              <UIcon name="i-lucide-loader-circle" class="size-4 animate-spin" /> Loading repos…
            </div>
            <div v-else class="max-h-44 overflow-y-auto rounded-xl border border-default divide-y divide-default">
              <button v-for="repo in ghRepos" :key="repo.full_name" type="button"
                class="w-full flex items-center justify-between px-3 py-2 text-sm hover:bg-elevated transition-colors text-left"
                :class="newProject.repository_url === repo.clone_url ? 'bg-primary/10 text-primary' : ''"
                @click="selectRepo(repo)">
                <span class="flex items-center gap-2">
                  <UIcon :name="repo.private ? 'i-lucide-lock' : 'i-lucide-book-open'" class="size-3.5 text-muted shrink-0" />
                  {{ repo.full_name }}
                </span>
                <UBadge size="xs" variant="subtle" color="neutral">{{ repo.default_branch }}</UBadge>
              </button>
            </div>
            <p v-if="newProject.repository_url" class="text-xs text-muted font-mono truncate">{{ newProject.repository_url }}</p>
          </div>
          <div v-else class="rounded-xl border border-default bg-elevated/40 p-4 flex items-start gap-3">
            <UIcon name="i-lucide-plug-zap" class="size-4 text-muted shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium">GitHub App not connected</p>
              <p class="text-xs text-muted mt-0.5">Go to <NuxtLink to="/settings/source-control" class="text-primary underline" @click="isNewModalOpen = false">Source Control</NuxtLink> to connect.</p>
            </div>
          </div>
        </template>

        <UFormField v-if="sourceMode === 'git_url'" label="Repository URL" name="repository_url">
          <UInput v-model="newProject.repository_url" placeholder="https://github.com/user/repo" class="w-full" />
        </UFormField>
        <UFormField v-if="sourceMode !== 'upload'" label="Branch" name="branch" hint="Type any branch name or pick a common one">
          <UInputMenu v-model="newProject.branch" :items="['main', 'master', 'develop', 'dev', 'staging', 'production']" placeholder="main" class="w-full" create-option />
        </UFormField>
        <UFormField label="Project Name" name="name">
          <UInput v-model="newProject.name" placeholder="my-awesome-app" class="w-full" />
        </UFormField>
        <UFormField label="Container Port" name="container_port">
          <UInput v-model="newProject.container_port" type="number" placeholder="3000" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2 pt-2">
          <UButton variant="ghost" color="neutral" @click="isNewModalOpen = false">Cancel</UButton>
          <UButton type="submit" :loading="creating" icon="i-lucide-rocket">Create Project</UButton>
        </div>
      </UForm>

      <!-- ── Track: Service — template grid ── -->
      <div v-else-if="track === 'service' && serviceStep === 'grid'" class="p-5 space-y-4">
        <div class="flex items-center gap-2">
          <UButton size="xs" variant="ghost" color="neutral" icon="i-lucide-arrow-left" @click="track = null" />
          <span class="text-xs text-muted">Back</span>
        </div>
        <div v-if="!templates.length" class="flex items-center justify-center py-10">
          <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin text-muted" />
        </div>
        <div v-else class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <button
            v-for="t in templates"
            :key="t.id"
            class="flex flex-col rounded-xl border border-default hover:border-primary/40 hover:bg-elevated/60 transition-all text-center group overflow-hidden"
            @click="selectTemplate(t)"
          >
            <div class="flex items-center justify-center bg-elevated/40 group-hover:bg-elevated transition-colors pt-5 pb-4 px-4">
              <img :src="templateImages[t.id]" :alt="t.name" class="w-14 h-14 object-contain drop-shadow-md" />
            </div>
            <div class="px-3 py-2.5">
              <p class="text-sm font-semibold text-highlighted">{{ t.name }}</p>
              <p class="text-xs text-muted leading-tight mt-0.5">{{ t.description }}</p>
            </div>
          </button>
        </div>
        <div class="flex justify-end">
          <UButton variant="ghost" color="neutral" @click="isNewModalOpen = false">Cancel</UButton>
        </div>
      </div>

      <!-- ── Track: Service — configure form ── -->
      <div v-else-if="track === 'service' && serviceStep === 'form' && selectedTemplate" class="p-5 space-y-4">
        <div class="flex items-center gap-2">
          <UButton size="xs" variant="ghost" color="neutral" icon="i-lucide-arrow-left" @click="serviceStep = 'grid'" />
          <span class="text-xs text-muted">Back to templates</span>
        </div>

        <!-- Template header -->
        <div class="flex items-center gap-3 p-3 rounded-xl bg-elevated/60 border border-default">
          <img :src="templateImages[selectedTemplate.id]" :alt="selectedTemplate.name" class="size-8 object-contain" />
          <div>
            <p class="font-semibold text-sm text-highlighted">{{ selectedTemplate.name }}</p>
            <p class="text-xs text-muted">{{ selectedTemplate.description }}</p>
          </div>
        </div>

        <!-- Name + image version -->
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-highlighted">Service Name</label>
          <UInput v-model="serviceName" :placeholder="`my-${selectedTemplate.id}`" class="w-full" />
        </div>
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-highlighted">Image Version</label>
          <USelectMenu v-model="serviceImage" :items="selectedTemplate.versions" class="w-full" />
        </div>

        <!-- Env var fields -->
        <div v-if="serviceEnvVars.length" class="space-y-3">
          <label class="text-sm font-medium text-highlighted">Configuration</label>
          <div v-for="(ev, i) in serviceEnvVars" :key="ev.key" class="space-y-1">
            <label class="text-xs text-muted font-mono">{{ ev.key }}</label>
            <UInput
              v-model="serviceEnvVars[i].value"
              :type="ev.secret ? 'password' : 'text'"
              :placeholder="selectedTemplate.env_var_fields[i]?.placeholder ?? ''"
              class="w-full"
            />
          </div>
        </div>

        <!-- Volume toggle -->
        <div v-if="selectedTemplate.volume_mount" class="flex items-center justify-between p-3 rounded-xl bg-elevated/60 border border-default">
          <div>
            <p class="text-sm font-medium text-highlighted">Persistent Storage</p>
            <p class="text-xs text-muted">Data survives container restarts</p>
          </div>
          <UToggle v-model="serviceVolumeEnabled" />
        </div>

        <div class="flex justify-end gap-2 pt-1">
          <UButton variant="ghost" color="neutral" @click="isNewModalOpen = false">Cancel</UButton>
          <UButton :loading="serviceCreating" icon="i-lucide-rocket" color="success" @click="submitService">
            Deploy {{ selectedTemplate.name }}
          </UButton>
        </div>
      </div>

    </template>
  </UModal>

  <!-- Delete confirm modal -->
  <UModal v-model:open="deleteConfirmOpen" title="Delete Project">
    <template #body>
      <div class="p-4 space-y-4">
        <div class="flex items-start gap-3">
          <div class="p-2 rounded-xl bg-red-500/10 shrink-0">
            <UIcon name="i-lucide-triangle-alert" class="size-5 text-red-400" />
          </div>
          <div>
            <p class="text-sm font-medium text-highlighted">Delete "{{ projectToDelete?.name }}"?</p>
            <p class="text-sm text-muted mt-1">This will stop and permanently remove the container, image, and all deployment history. This cannot be undone.</p>
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-1">
          <UButton variant="ghost" color="neutral" @click="deleteConfirmOpen = false">Cancel</UButton>
          <UButton color="error" icon="i-lucide-trash-2" @click="executeDelete">Delete Project</UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
