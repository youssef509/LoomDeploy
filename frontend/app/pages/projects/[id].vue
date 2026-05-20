<script setup lang="ts">
import type { Project, Deployment, EnvVar, ContainerStats, DomainList, ProjectDomain } from '~/types'
import { z } from 'zod'

definePageMeta({ middleware: 'auth' })

const route = useRoute()
const api = useApi()
const toast = useToast()
const auth = useAuthStore()

const projectId = route.params.id as string

const { data: project, pending, refresh } = useAsyncData(`project-${projectId}`, () => api.getProject(projectId))
const { data: deployments, refresh: refreshDeployments } = useAsyncData(`deployments-${projectId}`, () => api.getDeployments(projectId), { default: () => [] })
const { data: envVars, refresh: refreshEnv } = useAsyncData(`env-${projectId}`, () => api.getEnvVars(projectId), { default: () => [] })

function formatDuration(started: string, finished?: string) {
  if (!finished) return ''
  const total = Math.round((new Date(finished).getTime() - new Date(started).getTime()) / 1000)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  if (h > 0) return `${h}h ${m}m ${s}s`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

const statusColor = (status?: string) => {
  if (status === 'running') return 'success'
  if (status === 'failed') return 'error'
  if (status === 'building') return 'warning'
  if (status === 'stopped') return 'neutral'
  return 'neutral'
}

// ── Container stats ─────────────────────────────────────────────────────
const containerStats = ref<ContainerStats | null>(null)
let statsTimer: ReturnType<typeof setInterval> | null = null

async function refreshStats() {
  const res = await api.getContainerStats(projectId)
  containerStats.value = 'cpu_percent' in res ? res as ContainerStats : null
}

function startStatsPolling() {
  if (statsTimer) return
  refreshStats()
  statsTimer = setInterval(refreshStats, 5000)
}

function stopStatsPolling() {
  if (statsTimer) { clearInterval(statsTimer); statsTimer = null }
}

watch(() => project.value?.latest_deployment?.status, (status) => {
  if (status === 'running') startStatsPolling()
  else stopStatsPolling()
}, { immediate: true })

// ── Polling ──────────────────────────────────────────────────────────────
const deploying = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    await Promise.all([refresh(), refreshDeployments()])
    const status = project.value?.latest_deployment?.status
    if (status === 'running' || status === 'failed') stopPolling()
  }, 3000)
}

function stopPolling() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}

const isBuilding = computed(() => {
  const s = project.value?.latest_deployment?.status
  return s === 'building' || s === 'pending'
})

// Auto-start polling if we land on the page mid-build (e.g. webhook-triggered deploy)
watch(() => project.value?.latest_deployment?.status, (status) => {
  if ((status === 'building' || status === 'pending') && !pollTimer) {
    startPolling()
  }
}, { immediate: true })

// ── Log modal ─────────────────────────────────────────────────────────────
const logModalOpen = ref(false)
const logModalTitle = ref('')
const logContent = ref('')
const logStreaming = ref(false)
const logTerminal = ref<HTMLElement | null>(null)
const runtimeTerminal = ref<HTMLElement | null>(null)

watch(logContent, async () => {
  await nextTick()
  if (logTerminal.value) logTerminal.value.scrollTop = logTerminal.value.scrollHeight
})

function openStoredLogs(dep: Deployment) {
  logModalTitle.value = `Build Logs — ${dep.id.slice(0, 8)}`
  logContent.value = dep.build_logs ?? 'No logs available.'
  logStreaming.value = false
  logModalOpen.value = true
}

function parseSSEChunk(chunk: string): { lines: string[], done: boolean } {
  const lines: string[] = []
  let done = false
  for (const raw of chunk.split('\n')) {
    if (raw.startsWith('data: ')) lines.push(raw.slice(6))
    if (raw.startsWith('event: done')) done = true
  }
  return { lines, done }
}

async function streamBuildLogs() {
  logModalTitle.value = 'Live Build Logs'
  logContent.value = ''
  logStreaming.value = true
  logModalOpen.value = true

  try {
    const res = await fetch(`/api/projects/${projectId}/build-logs/stream`, {
      headers: { Authorization: `Bearer ${auth.token ?? ''}` }
    })
    if (!res.ok || !res.body) { logStreaming.value = false; return }

    const reader = res.body.getReader()
    const decoder = new TextDecoder()

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      const { lines, done: streamDone } = parseSSEChunk(decoder.decode(value, { stream: true }))
      for (const line of lines) logContent.value += line + '\n'
      if (streamDone) break
    }
  }
  catch { /* stream cancelled when modal closes */ }
  finally { logStreaming.value = false }
}

// ── Runtime logs ──────────────────────────────────────────────────────────
const runtimeLogs = ref('')
const runtimeStreaming = ref(false)

watch(runtimeLogs, async () => {
  await nextTick()
  if (runtimeTerminal.value) runtimeTerminal.value.scrollTop = runtimeTerminal.value.scrollHeight
})
let runtimeReader: ReadableStreamDefaultReader<Uint8Array> | null = null

async function startRuntimeLogs() {
  runtimeLogs.value = ''
  runtimeStreaming.value = true
  try {
    const res = await fetch(`/api/projects/${projectId}/logs`, {
      headers: { Authorization: `Bearer ${auth.token ?? ''}` }
    })
    if (!res.ok || !res.body) { runtimeStreaming.value = false; return }
    runtimeReader = res.body.getReader()
    const decoder = new TextDecoder()
    while (true) {
      const { done, value } = await runtimeReader.read()
      if (done) break
      const chunk = decoder.decode(value, { stream: true })
      for (const raw of chunk.split('\n')) {
        if (raw.startsWith('data: ')) runtimeLogs.value += raw.slice(6) + '\n'
      }
    }
  }
  catch { /* cancelled */ }
  finally { runtimeStreaming.value = false }
}

function stopRuntimeLogs() {
  runtimeReader?.cancel()
  runtimeReader = null
  runtimeStreaming.value = false
}

// ── Deploy ────────────────────────────────────────────────────────────────
async function deploy() {
  deploying.value = true
  try {
    await api.deployProject(projectId)
    toast.add({ title: 'Deployment started', description: 'Your app is building...', color: 'success' })
    await refresh()       // immediately reflect pending → disables Deploy button via isBuilding
    startPolling()
    streamBuildLogs()
  }
  catch (err: any) {
    toast.add({ title: 'Deploy failed', description: err?.data?.message, color: 'error' })
  }
  finally {
    deploying.value = false
  }
}

function downloadLogs() {
  if (!logContent.value) return
  const blob = new Blob([logContent.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${logModalTitle.value.replace(/\s+/g, '_')}.txt`
  a.click()
  URL.revokeObjectURL(url)
}

const deletingDepId = ref<string | null>(null)
const expandedLogDep = ref<string | null>(null)

function lastLines(logs: string, n: number): string {
  return logs.split('\n').filter(l => l.trim()).slice(-n).join('\n')
}

async function deleteDeployment(dep: Deployment) {
  deletingDepId.value = dep.id
  try {
    await api.deleteDeployment(projectId, dep.id)
    toast.add({ title: 'Deployment deleted', color: 'success' })
    refreshDeployments()
  } catch (err: any) {
    toast.add({ title: 'Delete failed', description: err?.data?.message, color: 'error' })
  } finally {
    deletingDepId.value = null
  }
}

async function redeployFrom(dep: Deployment) {
  deploying.value = true
  try {
    await api.redeployDeployment(projectId, dep.id)
    toast.add({ title: `Rolling back to ${dep.commit_sha?.slice(0, 7)}`, description: 'Deployment started', color: 'success' })
    await refresh()
    startPolling()
    streamBuildLogs()
  } catch (err: any) {
    toast.add({ title: 'Rollback failed', description: err?.data?.message, color: 'error' })
  } finally {
    deploying.value = false
  }
}

// ── Container actions ─────────────────────────────────────────────────────
const actionLoading = ref<string | null>(null)
async function containerAction(action: 'start' | 'stop' | 'restart') {
  actionLoading.value = action
  try {
    await api.containerAction(projectId, action)
    toast.add({ title: `Container ${action}ed`, color: 'success' })
    if (action === 'stop') stopRuntimeLogs()
    setTimeout(refresh, 1500)
  }
  catch (err: any) {
    toast.add({ title: `${action} failed`, description: err?.data?.message, color: 'error' })
  }
  finally {
    actionLoading.value = null
  }
}

// ── Env vars ──────────────────────────────────────────────────────────────
const envSchema = z.object({ key: z.string().min(1), value: z.string() })
const newEnv = reactive({ key: '', value: '' })
const savingEnv = ref(false)

async function addEnvVar() {
  if (!envSchema.safeParse(newEnv).success) return
  savingEnv.value = true
  try {
    const current = (envVars.value as EnvVar[]).map(e => ({ key: e.key, value: e.value }))
    await api.setEnvVars(projectId, [...current, { key: newEnv.key, value: newEnv.value }])
    newEnv.key = ''
    newEnv.value = ''
    refreshEnv()
    toast.add({ title: 'Env var saved', color: 'success' })
  }
  catch { toast.add({ title: 'Failed to save', color: 'error' }) }
  finally { savingEnv.value = false }
}

async function removeEnvVar(key: string) {
  const updated = (envVars.value as EnvVar[]).filter(e => e.key !== key).map(e => ({ key: e.key, value: e.value }))
  try {
    await api.setEnvVars(projectId, updated)
    refreshEnv()
    toast.add({ title: `"${key}" removed`, color: 'success' })
  }
  catch { toast.add({ title: 'Failed to remove', color: 'error' }) }
}

// ── Settings (edit project) ───────────────────────────────────────────────
const settingsForm = reactive({
  name: '',
  domain: '',
  repository_url: '',
  branch: '',
  container_port: 3000,
  git_token: '',
  notification_webhook_url: '',
  cpu_limit: 0,
  memory_limit_mb: 0,
  health_check_url: '',
  dockerfile_content: ''
})
const savingSettings = ref(false)

watch(project, (p) => {
  if (!p) return
  settingsForm.name = p.name
  settingsForm.domain = p.domain
  settingsForm.repository_url = p.repository_url ?? ''
  settingsForm.branch = p.branch ?? ''
  settingsForm.container_port = p.container_port
  settingsForm.git_token = ''
  settingsForm.notification_webhook_url = p.notification_webhook_url ?? ''
  settingsForm.cpu_limit = p.cpu_limit ?? 0
  settingsForm.memory_limit_mb = p.memory_limit_mb ?? 0
  settingsForm.health_check_url = p.health_check_url ?? ''
  settingsForm.dockerfile_content = p.dockerfile_content ?? ''
}, { immediate: true })

async function saveSettings() {
  savingSettings.value = true
  try {
    const payload: Record<string, unknown> = {
      name: settingsForm.name,
      repository_url: settingsForm.repository_url,
      branch: settingsForm.branch,
      container_port: Number(settingsForm.container_port),
      notification_webhook_url: settingsForm.notification_webhook_url,
      cpu_limit: Number(settingsForm.cpu_limit),
      memory_limit_mb: Number(settingsForm.memory_limit_mb),
      health_check_url: settingsForm.health_check_url,
      dockerfile_content: settingsForm.dockerfile_content
    }
    if (settingsForm.git_token !== '') {
      payload.git_token = settingsForm.git_token
    }
    await api.updateProject(projectId, payload as any)
    toast.add({ title: 'Settings saved', description: 'Domain/port changes apply on next deployment.', color: 'success' })
    settingsForm.git_token = ''
    refresh()
  } catch (err: any) {
    toast.add({ title: 'Save failed', description: err?.data?.message, color: 'error' })
  } finally {
    savingSettings.value = false
  }
}

async function clearToken() {
  savingSettings.value = true
  try {
    await api.updateProject(projectId, { git_token: '' } as any)
    toast.add({ title: 'Token removed', color: 'success' })
    refresh()
  } catch (err: any) {
    toast.add({ title: 'Failed to remove token', description: err?.data?.message, color: 'error' })
  } finally {
    savingSettings.value = false
  }
}

// ── Webhook ──────────────────────────────────────────────────────────────
const webhookUrl = computed(() => {
  const secret = project.value?.webhook_secret
  if (!secret) return ''
  return `${window.location.origin}/api/webhooks/${secret}`
})

const regeneratingWebhook = ref(false)
async function doRegenerateWebhook() {
  if (!confirm('Regenerating will invalidate the current webhook URL. Any existing webhooks on GitHub/GitLab will need to be updated. Continue?')) return
  regeneratingWebhook.value = true
  try {
    await api.regenerateWebhook(projectId)
    toast.add({ title: 'Webhook URL regenerated', color: 'success' })
    refresh()
  } catch (err: any) {
    toast.add({ title: 'Failed to regenerate', description: err?.data?.message, color: 'error' })
  } finally {
    regeneratingWebhook.value = false
  }
}

function copyWebhookUrl() {
  if (!webhookUrl.value) return
  navigator.clipboard.writeText(webhookUrl.value)
  toast.add({ title: 'Webhook URL copied', color: 'success' })
}

// ── Domain management ───────────────────────────────────────────────────
const domainList = ref<DomainList | null>(null)
const newDomain = ref('')
const addingDomain = ref(false)
const removingDomainId = ref<string | null>(null)
const updatingPrimary = ref(false)
const editPrimaryDomain = ref('')
const showEditPrimary = ref(false)

async function loadDomains() {
  try { domainList.value = await api.listDomains(projectId) } catch { /* ignore */ }
}

async function doAddDomain() {
  if (!newDomain.value.trim()) return
  addingDomain.value = true
  try {
    await api.addDomain(projectId, newDomain.value.trim())
    newDomain.value = ''
    await loadDomains()
    toast.add({ title: 'Domain added', description: 'Redeploy to activate the new domain.', color: 'success' })
  } catch (err: any) {
    toast.add({ title: 'Failed to add domain', description: err?.data?.message, color: 'error' })
  } finally {
    addingDomain.value = false
  }
}

async function doRemoveDomain(d: ProjectDomain) {
  removingDomainId.value = d.id
  try {
    await api.removeDomain(projectId, d.id)
    await loadDomains()
    toast.add({ title: 'Domain removed', color: 'success' })
  } catch (err: any) {
    toast.add({ title: 'Failed to remove domain', description: err?.data?.message, color: 'error' })
  } finally {
    removingDomainId.value = null
  }
}

async function doUpdatePrimary() {
  if (!editPrimaryDomain.value.trim()) return
  updatingPrimary.value = true
  try {
    await api.updatePrimaryDomain(projectId, editPrimaryDomain.value.trim())
    showEditPrimary.value = false
    await Promise.all([refresh(), loadDomains()])
    toast.add({ title: 'Primary domain updated', description: 'Redeploy to activate.', color: 'success' })
  } catch (err: any) {
    toast.add({ title: 'Failed to update domain', description: err?.data?.message, color: 'error' })
  } finally {
    updatingPrimary.value = false
  }
}

// ── Cleanup ─────────────────────────────────────────────────────────────
onUnmounted(() => {
  stopPolling()
  stopRuntimeLogs()
  stopStatsPolling()
})

const activeTab = ref('overview')
watch(() => activeTab.value, (tab) => {
  if (tab === 'settings') loadDomains()
})
const tabs = [
  { label: 'Overview', slot: 'overview', icon: 'i-lucide-layout-dashboard', value: 'overview' },
  { label: 'Deployments', slot: 'deployments', icon: 'i-lucide-rocket', value: 'deployments' },
  { label: 'Env Variables', slot: 'env', icon: 'i-lucide-key-round', value: 'env' },
  { label: 'Runtime Logs', slot: 'runtime', icon: 'i-lucide-terminal', value: 'runtime' },
  { label: 'Settings', slot: 'settings', icon: 'i-lucide-settings-2', value: 'settings' }
]
</script>

<template>
  <UDashboardPanel :id="`project-${projectId}`">
    <template #header>
      <UDashboardNavbar :title="project?.name ?? 'Project'">
        <template #leading>
          <UButton variant="ghost" color="neutral" size="sm" icon="i-lucide-arrow-left" to="/projects" />
          <UDashboardSidebarCollapse class="!text-muted hover:!text-highlighted" />
        </template>
        <template #right>
          <UBadge
            v-if="project?.latest_deployment"
            :color="statusColor(project.latest_deployment.status)"
            variant="subtle"
            class="font-mono"
          >
            {{ project.latest_deployment.status }}
          </UBadge>

          <UButton
            v-if="project?.latest_deployment?.status === 'running'"
            variant="outline" size="sm" icon="i-lucide-square"
            :loading="actionLoading === 'stop'" color="error"
            @click="containerAction('stop')"
          >Stop</UButton>
          <UButton
            v-else-if="project?.latest_deployment?.status === 'stopped'"
            variant="outline" size="sm" icon="i-lucide-play"
            :loading="actionLoading === 'start'" color="success"
            @click="containerAction('start')"
          >Start</UButton>

          <UButton
            v-if="project?.latest_deployment?.status === 'running'"
            variant="outline" size="sm" icon="i-lucide-rotate-cw"
            :loading="actionLoading === 'restart'" color="neutral"
            @click="containerAction('restart')"
          >Restart</UButton>

          <UButton
            size="sm" icon="i-lucide-rocket"
            :loading="deploying"
            :disabled="isBuilding"
            @click="deploy"
          >
            {{ isBuilding ? 'Building…' : 'Deploy' }}
          </UButton>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <!-- Spinner only on INITIAL load — not during background refreshes (avoids tab reset) -->
      <div v-if="pending && !project" class="flex justify-center py-16">
        <UIcon name="i-lucide-loader-circle" class="size-8 text-muted animate-spin" />
      </div>

      <div v-else-if="!project" class="flex flex-col items-center justify-center py-24 gap-3">
        <UIcon name="i-lucide-file-question" class="size-10 text-muted" />
        <p class="text-sm text-muted">Project not found.</p>
        <UButton to="/projects" variant="outline" size="sm">Back to projects</UButton>
      </div>

      <div v-else class="p-6">
        <UTabs v-model="activeTab" :items="tabs" class="w-full">

          <!-- ── Overview ── -->
          <template #overview>
            <div class="mt-6 grid grid-cols-1 md:grid-cols-2 gap-4">
              <UCard>
                <template #header><h3 class="text-sm font-semibold">Project Details</h3></template>
                <dl class="space-y-3 text-sm">
                  <div class="flex justify-between">
                    <dt class="text-muted">Domain</dt>
                    <dd>
                      <a :href="`https://${project.domain}`" target="_blank" class="font-mono text-primary hover:underline flex items-center gap-1">
                        {{ project.domain }}<UIcon name="i-lucide-external-link" class="size-3" />
                      </a>
                    </dd>
                  </div>
                  <div class="flex justify-between">
                    <dt class="text-muted">Source</dt>
                    <dd class="flex items-center gap-1.5">
                      <UIcon :name="project.source_type === 'git' ? 'i-simple-icons-git' : 'i-lucide-upload'" class="size-4" />
                      {{ project.source_type }}
                    </dd>
                  </div>
                  <div v-if="project.repository_url" class="flex justify-between">
                    <dt class="text-muted">Repository</dt>
                    <dd class="font-mono text-xs truncate max-w-[200px]">{{ project.repository_url }}</dd>
                  </div>
                  <div v-if="project.branch" class="flex justify-between">
                    <dt class="text-muted">Branch</dt>
                    <dd><UBadge variant="soft" size="xs" icon="i-lucide-git-branch">{{ project.branch }}</UBadge></dd>
                  </div>
                  <div class="flex justify-between">
                    <dt class="text-muted">Container Port</dt>
                    <dd class="font-mono">{{ project.container_port }}</dd>
                  </div>
                  <div class="flex justify-between">
                    <dt class="text-muted">Created</dt>
                    <dd class="text-muted">{{ new Date(project.created_at).toLocaleDateString() }}</dd>
                  </div>
                </dl>
              </UCard>

              <!-- Resource Usage -->
              <UCard v-if="containerStats">
                <template #header><h3 class="text-sm font-semibold">Resource Usage</h3></template>
                <dl class="space-y-3 text-sm">
                  <div>
                    <div class="flex justify-between mb-1">
                      <dt class="text-muted">CPU</dt>
                      <dd class="font-mono">{{ containerStats.cpu_percent.toFixed(2) }}%</dd>
                    </div>
                    <div class="w-full bg-elevated rounded-full h-1.5">
                      <div class="bg-primary rounded-full h-1.5 transition-all" :style="{ width: Math.min(containerStats.cpu_percent, 100) + '%' }" />
                    </div>
                  </div>
                  <div>
                    <div class="flex justify-between mb-1">
                      <dt class="text-muted">Memory</dt>
                      <dd class="font-mono">{{ containerStats.memory_usage_mb.toFixed(0) }} / {{ containerStats.memory_limit_mb.toFixed(0) }} MB</dd>
                    </div>
                    <div class="w-full bg-elevated rounded-full h-1.5">
                      <div
                        class="rounded-full h-1.5 transition-all"
                        :class="containerStats.memory_percent > 80 ? 'bg-error' : containerStats.memory_percent > 60 ? 'bg-warning' : 'bg-success'"
                        :style="{ width: Math.min(containerStats.memory_percent, 100) + '%' }"
                      />
                    </div>
                  </div>
                </dl>
              </UCard>

              <UCard>
                <template #header><h3 class="text-sm font-semibold">Latest Deployment</h3></template>
                <div v-if="!project.latest_deployment" class="py-6 text-center text-sm text-muted">
                  <UIcon name="i-lucide-rocket" class="size-8 mx-auto mb-2 opacity-30" />
                  <p>Never deployed</p>
                  <UButton size="xs" class="mt-3" @click="deploy">Deploy Now</UButton>
                </div>
                <dl v-else class="space-y-3 text-sm">
                  <div class="flex justify-between">
                    <dt class="text-muted">Status</dt>
                    <dd>
                      <UBadge :color="statusColor(project.latest_deployment.status)" variant="subtle">
                        {{ project.latest_deployment.status }}
                      </UBadge>
                    </dd>
                  </div>
                  <div v-if="project.latest_deployment.container_id" class="flex justify-between">
                    <dt class="text-muted">Container</dt>
                    <dd class="font-mono text-xs">{{ project.latest_deployment.container_id.slice(0, 12) }}</dd>
                  </div>
                  <div class="flex justify-between">
                    <dt class="text-muted">Started</dt>
                    <dd class="text-muted">{{ new Date(project.latest_deployment.started_at).toLocaleString() }}</dd>
                  </div>
                </dl>
              </UCard>
            </div>
          </template>

          <!-- ── Deployments ── -->
          <template #deployments>
            <div class="mt-6 space-y-3">
              <div v-if="!(deployments as Deployment[]).length" class="py-12 text-center text-sm text-muted">
                No deployments yet. Click <strong>Deploy</strong> to get started.
              </div>
              <div
                v-for="dep in (deployments as Deployment[])"
                :key="dep.id"
                class="rounded-xl border overflow-hidden transition-colors"
                :class="dep.status === 'failed' ? 'border-error/30 bg-error/5' : 'border-default bg-elevated/50 hover:bg-elevated'"
              >
                <!-- Main row -->
                <div class="flex items-center justify-between p-4">
                  <div class="flex items-center gap-3">
                    <UBadge :color="statusColor(dep.status)" variant="subtle">{{ dep.status }}</UBadge>
                    <UBadge v-if="dep.status === 'running'" color="success" variant="soft" size="xs">LIVE</UBadge>
                    <div>
                      <p class="text-xs font-mono text-muted">
                        {{ dep.id.slice(0, 8) }}
                        <span v-if="dep.commit_sha" class="ml-1.5 text-primary">@ {{ dep.commit_sha.slice(0, 7) }}</span>
                      </p>
                      <p class="text-xs text-muted">
                        {{ new Date(dep.started_at).toLocaleString() }}
                        <span v-if="dep.finished_at" class="ml-1.5 font-mono">({{ formatDuration(dep.started_at, dep.finished_at) }})</span>
                      </p>
                    </div>
                  </div>
                  <div class="flex items-center gap-2">
                    <!-- Live logs button for active builds -->
                    <UButton
                      v-if="dep.status === 'building' || dep.status === 'pending'"
                      variant="ghost" size="xs" icon="i-lucide-radio" color="warning"
                      class="animate-pulse"
                      @click="streamBuildLogs()"
                    >Live Logs</UButton>
                    <!-- Stored logs for finished builds -->
                    <UButton
                      v-else-if="dep.build_logs"
                      variant="ghost" size="xs" icon="i-lucide-terminal" color="neutral"
                      @click="openStoredLogs(dep)"
                    >Logs</UButton>
                    <UTooltip :text="isBuilding ? 'A build is already in progress' : dep.commit_sha ? `Rollback to ${dep.commit_sha.slice(0,7)}` : 'Redeploy from current HEAD'">
                      <UButton
                        variant="ghost" size="xs" icon="i-lucide-rotate-cw" color="neutral"
                        :disabled="isBuilding"
                        :loading="deploying"
                        @click="dep.commit_sha ? redeployFrom(dep) : deploy()"
                      >{{ dep.commit_sha ? 'Rollback' : 'Redeploy' }}</UButton>
                    </UTooltip>
                    <UTooltip v-if="dep.status === 'failed' || dep.status === 'stopped'" text="Delete this deployment record">
                      <UButton
                        variant="ghost" size="xs" icon="i-lucide-trash-2" color="error"
                        :loading="deletingDepId === dep.id"
                        @click="deleteDeployment(dep)"
                      />
                    </UTooltip>
                  </div>
                </div>
                <!-- Inline error excerpt for failed deployments -->
                <div v-if="dep.status === 'failed' && dep.build_logs" class="border-t border-error/20">
                  <button
                    class="w-full flex items-center gap-1.5 px-4 py-1.5 text-xs text-error/70 hover:text-error transition-colors text-left"
                    @click="expandedLogDep = expandedLogDep === dep.id ? null : dep.id"
                  >
                    <UIcon name="i-lucide-chevron-right" class="size-3 transition-transform duration-200" :class="expandedLogDep === dep.id ? 'rotate-90' : ''" />
                    {{ expandedLogDep === dep.id ? 'Hide error details' : 'View error details' }}
                  </button>
                  <pre v-if="expandedLogDep === dep.id" class="px-4 pb-3 text-xs font-mono text-error/80 whitespace-pre-wrap leading-relaxed overflow-auto max-h-48">{{ lastLines(dep.build_logs, 10) }}</pre>
                </div>
              </div>
            </div>
          </template>

          <!-- ── Env Variables ── -->
          <template #env>
            <div class="mt-6 space-y-4">
              <UCard>
                <template #header>
                  <h3 class="text-sm font-semibold">Environment Variables</h3>
                  <p class="text-xs text-muted mt-0.5">Changes apply on next deployment</p>
                </template>
                <div class="space-y-2">
                  <div
                    v-for="env in (envVars as EnvVar[])"
                    :key="env.key"
                    class="flex items-center gap-2"
                  >
                    <UInput :model-value="env.key" readonly class="flex-1 font-mono text-xs" />
                    <UInput :model-value="env.value" type="password" readonly class="flex-1 font-mono text-xs" />
                    <UButton variant="ghost" size="xs" icon="i-lucide-trash-2" color="error" @click="removeEnvVar(env.key)" />
                  </div>
                </div>
                <div class="flex items-center gap-2 mt-4 pt-4 border-t border-default">
                  <UInput v-model="newEnv.key" placeholder="KEY" class="flex-1 font-mono text-xs" />
                  <UInput v-model="newEnv.value" placeholder="value" class="flex-1 font-mono text-xs" />
                  <UButton size="xs" icon="i-lucide-plus" :loading="savingEnv" :disabled="!newEnv.key" @click="addEnvVar">Add</UButton>
                </div>
              </UCard>
            </div>
          </template>

          <!-- ── Settings ── -->
          <template #settings>
            <div class="mt-6 grid grid-cols-1 xl:grid-cols-2 gap-4 items-start">
              <div class="space-y-4">
              <UCard>
                <template #header>
                  <div>
                    <h3 class="text-sm font-semibold">Project Settings</h3>
                    <p class="text-xs text-muted mt-0.5">Branch and port changes take effect on next deployment.</p>
                  </div>
                </template>

                <div class="space-y-4">
                  <UFormField label="Project Name" name="name">
                    <UInput v-model="settingsForm.name" placeholder="my-app" class="w-full" />
                  </UFormField>

                  <template v-if="project?.source_type === 'git'">
                    <UFormField label="Branch" name="branch" hint="Type any branch name or pick a common one">
                      <UInputMenu
                        v-model="settingsForm.branch"
                        :items="['main', 'master', 'develop', 'dev', 'staging', 'production']"
                        placeholder="main"
                        class="w-full"
                        create-option
                      />
                    </UFormField>
                  </template>

                  <UFormField label="Container Port" name="container_port">
                    <UInput v-model="settingsForm.container_port" type="number" placeholder="3000" class="w-full" />
                  </UFormField>

                  <UFormField
                    label="Dockerfile Override"
                    hint="Leave empty to use auto-detection. Paste a custom Dockerfile to override."
                  >
                    <UTextarea
                      v-model="settingsForm.dockerfile_content"
                      placeholder="FROM node:20-alpine&#10;WORKDIR /app&#10;..."
                      :rows="6"
                      class="w-full font-mono text-xs"
                    />
                  </UFormField>
                </div>

                <USeparator class="my-4" />

                <div class="space-y-4">
                  <div>
                    <p class="text-sm font-semibold mb-0.5">Deploy Notifications</p>
                    <p class="text-xs text-muted">Get notified on Discord or Slack when a deployment succeeds or fails.</p>
                  </div>
                  <UFormField label="Webhook URL" name="notification_webhook_url" hint="Discord / Slack / any webhook that accepts JSON POST with 'content' field">
                    <UInput v-model="settingsForm.notification_webhook_url" placeholder="https://discord.com/api/webhooks/..." class="w-full" />
                  </UFormField>
                </div>

                <template #footer>
                  <div class="flex justify-end">
                    <UButton icon="i-lucide-save" :loading="savingSettings" @click="saveSettings">Save Changes</UButton>
                  </div>
                </template>
              </UCard>

              <!-- Webhook Auto-Deploy -->
              <UCard>
                <template #header>
                  <div class="flex items-center gap-2">
                    <UIcon name="i-lucide-webhook" class="size-4" />
                    <div>
                      <h3 class="text-sm font-semibold">Webhook Auto-Deploy</h3>
                      <p class="text-xs text-muted mt-0.5">Push to <code class="bg-elevated px-1 rounded">{{ project?.branch ?? 'main' }}</code> → automatically triggers a deployment</p>
                    </div>
                  </div>
                </template>

                <div class="space-y-4">
                  <UFormField label="Webhook URL" hint="Add this URL to your git provider's webhook settings (push events only)">
                    <div class="flex gap-2">
                      <UInput :model-value="webhookUrl" readonly class="flex-1 font-mono text-xs" />
                      <UButton variant="outline" icon="i-lucide-copy" @click="copyWebhookUrl" />
                    </div>
                  </UFormField>

                  <!-- Setup instructions -->
                  <div class="rounded-lg bg-elevated/60 border border-default px-4 py-3 space-y-2 text-sm">
                    <p class="font-semibold text-xs uppercase tracking-wide text-highlighted">GitHub setup</p>
                    <ol class="space-y-1 text-muted">
                      <li>1. Go to your repo → <strong>Settings → Webhooks → Add webhook</strong></li>
                      <li>2. Paste the URL above as <strong>Payload URL</strong></li>
                      <li>3. Set content type to <code class="bg-default border border-default px-1 rounded text-xs">application/json</code></li>
                      <li>4. Select <strong>Just the push event</strong></li>
                      <li>5. Click <strong>Add webhook</strong></li>
                    </ol>
                    <p class="text-xs text-muted mt-1">For GitLab: use the same URL under <strong>Settings → Webhooks</strong>, enable Push events.</p>
                  </div>

                  <div class="flex justify-end">
                    <UButton
                      size="xs" color="error" variant="ghost" icon="i-lucide-rotate-cw"
                      :loading="regeneratingWebhook"
                      @click="doRegenerateWebhook"
                    >
                      Regenerate Secret
                    </UButton>
                  </div>
                </div>
              </UCard>

              </div>

              <!-- Right column: Domains + Resource Limits + Health Check -->
              <div class="space-y-4">

              <!-- Domains -->
              <UCard>
                <template #header>
                  <div class="flex items-center gap-2">
                    <UIcon name="i-lucide-globe" class="size-4" />
                    <div>
                      <h3 class="text-sm font-semibold">Domains</h3>
                      <p class="text-xs text-muted mt-0.5">Changes take effect on next deployment.</p>
                    </div>
                  </div>
                </template>

                <div class="space-y-4">
                  <!-- Primary domain -->
                  <div>
                    <p class="text-xs font-medium text-muted mb-1.5">Primary Domain</p>
                    <div class="flex items-center gap-2">
                      <div class="flex-1 flex items-center gap-2 rounded-md border border-default bg-elevated/60 px-3 py-2 min-w-0">
                        <UIcon name="i-lucide-link" class="size-3.5 shrink-0 text-muted" />
                        <span class="text-sm font-mono truncate">{{ domainList?.primary_domain ?? project?.domain }}</span>
                        <UBadge v-if="domainList?.is_generated ?? project?.is_generated_domain" size="xs" color="violet" variant="subtle" class="shrink-0">auto</UBadge>
                      </div>
                      <UButton
                        size="xs" variant="outline" icon="i-lucide-pencil"
                        @click="editPrimaryDomain = domainList?.primary_domain ?? project?.domain ?? ''; showEditPrimary = !showEditPrimary"
                      />
                      <UButton
                        size="xs" variant="ghost"
                        :icon="domainList?.primary_domain ? 'i-lucide-external-link' : 'i-lucide-external-link'"
                        :to="`https://${domainList?.primary_domain ?? project?.domain}`"
                        target="_blank"
                      />
                    </div>

                    <!-- Edit primary inline -->
                    <div v-if="showEditPrimary" class="mt-2 flex gap-2">
                      <UInput v-model="editPrimaryDomain" placeholder="app.yourdomain.com" class="flex-1 font-mono text-xs" @keyup.enter="doUpdatePrimary" />
                      <UButton size="xs" :loading="updatingPrimary" @click="doUpdatePrimary">Save</UButton>
                      <UButton size="xs" variant="ghost" color="neutral" @click="showEditPrimary = false">Cancel</UButton>
                    </div>
                  </div>

                  <!-- Extra domains list -->
                  <div v-if="domainList?.extra_domains?.length">
                    <p class="text-xs font-medium text-muted mb-1.5">Additional Domains</p>
                    <ul class="space-y-1.5">
                      <li
                        v-for="d in domainList.extra_domains"
                        :key="d.id"
                        class="flex items-center gap-2 rounded-md border border-default bg-elevated/40 px-3 py-1.5"
                      >
                        <UIcon name="i-lucide-link-2" class="size-3.5 text-muted shrink-0" />
                        <span class="flex-1 text-sm font-mono truncate">{{ d.domain }}</span>
                        <UButton
                          size="xs" variant="ghost" color="error" icon="i-lucide-x"
                          :loading="removingDomainId === d.id"
                          @click="doRemoveDomain(d)"
                        />
                      </li>
                    </ul>
                  </div>

                  <!-- Add extra domain -->
                  <div>
                    <p class="text-xs font-medium text-muted mb-1.5">Add Custom Domain</p>
                    <div class="flex gap-2">
                      <UInput
                        v-model="newDomain"
                        placeholder="store.yourdomain.com"
                        class="flex-1 font-mono text-xs"
                        @keyup.enter="doAddDomain"
                      />
                      <UButton size="sm" icon="i-lucide-plus" :loading="addingDomain" @click="doAddDomain">Add</UButton>
                    </div>
                  </div>

                  <!-- DNS hint -->
                  <div class="rounded-lg bg-elevated/60 border border-default px-3 py-2.5 text-xs text-muted space-y-0.5">
                    <p class="font-medium text-highlighted">DNS setup required for custom domains</p>
                    <p>Point an <strong>A record</strong> to your server IP, or a <strong>CNAME</strong> to your primary domain.</p>
                    <p>Traefik will auto-issue TLS certs via Let's Encrypt.</p>
                  </div>
                </div>
              </UCard>

              <!-- Resource Limits -->
              <UCard>
                <template #header>
                  <div class="flex items-center gap-2">
                    <UIcon name="i-lucide-cpu" class="size-4" />
                    <div>
                      <h3 class="text-sm font-semibold">Resource Limits</h3>
                      <p class="text-xs text-muted mt-0.5">0 = unlimited. Applied on next deployment.</p>
                    </div>
                  </div>
                </template>
                <div class="space-y-4">
                  <UFormField label="CPU Limit" hint="% of 1 CPU core. 100 = 1 core, 200 = 2 cores. 0 = unlimited.">
                    <div class="flex items-center gap-2">
                      <UInput v-model="settingsForm.cpu_limit" type="number" min="0" max="1600" placeholder="0" class="w-32" />
                      <span class="text-sm text-muted">%</span>
                    </div>
                  </UFormField>
                  <UFormField label="Memory Limit" hint="0 = unlimited. E.g. 512 = 512 MB.">
                    <div class="flex items-center gap-2">
                      <UInput v-model="settingsForm.memory_limit_mb" type="number" min="0" placeholder="0" class="w-32" />
                      <span class="text-sm text-muted">MB</span>
                    </div>
                  </UFormField>
                </div>
                <template #footer>
                  <div class="flex justify-end">
                    <UButton icon="i-lucide-save" :loading="savingSettings" @click="saveSettings">Save</UButton>
                  </div>
                </template>
              </UCard>

              <!-- Health Check -->
              <UCard>
                <template #header>
                  <div class="flex items-center gap-2">
                    <UIcon name="i-lucide-heart-pulse" class="size-4" />
                    <div>
                      <h3 class="text-sm font-semibold">Health Check</h3>
                      <p class="text-xs text-muted mt-0.5">Pinged every 30s — container auto-restarts after 3 consecutive failures</p>
                    </div>
                  </div>
                </template>
                <UFormField label="Health Check URL" hint="HTTP endpoint to GET, e.g. https://app.yourdomain.com/health">
                  <UInput v-model="settingsForm.health_check_url" placeholder="https://app.yourdomain.com/health" class="w-full" />
                </UFormField>
                <template #footer>
                  <div class="flex justify-end">
                    <UButton icon="i-lucide-save" :loading="savingSettings" @click="saveSettings">Save</UButton>
                  </div>
                </template>
              </UCard>

              </div>
            </div>
          </template>

          <!-- ── Runtime Logs ── -->
          <template #runtime>
            <div class="mt-6 flex flex-col gap-3">
              <div class="flex items-center gap-2">
                <UButton
                  v-if="!runtimeStreaming"
                  size="sm" icon="i-lucide-play" color="success" variant="outline"
                  :disabled="project.latest_deployment?.status !== 'running'"
                  @click="startRuntimeLogs"
                >Connect</UButton>
                <UButton
                  v-else
                  size="sm" icon="i-lucide-square" color="error" variant="outline"
                  @click="stopRuntimeLogs"
                >Disconnect</UButton>
                <UBadge v-if="runtimeStreaming" color="success" variant="subtle" class="animate-pulse">
                  <UIcon name="i-lucide-radio" class="size-3 mr-1" /> LIVE
                </UBadge>
                <span v-if="project.latest_deployment?.status !== 'running'" class="text-xs text-muted">
                  Container must be running to stream logs
                </span>
              </div>
              <pre
                ref="runtimeTerminal"
                class="h-[60vh] p-4 text-xs font-mono text-green-400 bg-black rounded-xl overflow-auto leading-relaxed whitespace-pre-wrap border border-default"
              >{{ runtimeLogs || 'No output yet. Click Connect to start streaming.' }}</pre>
            </div>
          </template>

        </UTabs>
      </div>
    </template>
  </UDashboardPanel>

  <!-- ── Log Modal (build logs: stored + live) ── -->
  <UModal v-model:open="logModalOpen" :title="logModalTitle" fullscreen>
    <template #header>
      <div class="flex items-center gap-3 flex-1">
        <span class="font-semibold text-sm">{{ logModalTitle }}</span>
        <UBadge v-if="logStreaming" color="success" variant="subtle" class="animate-pulse">
          <UIcon name="i-lucide-radio" class="size-3 mr-1" /> LIVE
        </UBadge>
        <div class="ml-auto flex items-center gap-2">
          <UButton
            v-if="logContent && !logStreaming"
            variant="ghost" size="xs" icon="i-lucide-download" color="neutral"
            @click="downloadLogs"
          >Download</UButton>
          <UButton variant="ghost" size="xs" icon="i-lucide-x" color="neutral" @click="logModalOpen = false" />
        </div>
      </div>
    </template>
    <template #body>
      <pre
        ref="logTerminal"
        class="p-4 text-xs font-mono text-green-400 bg-black h-[calc(100vh-3.5rem)] overflow-y-auto leading-relaxed whitespace-pre-wrap"
      >{{ logContent || 'Waiting for output...' }}</pre>
    </template>
  </UModal>
</template>
