<script setup lang="ts">
import type { Project, RecentDeployment } from '~/types'

definePageMeta({ middleware: 'auth' })
useDashboard()

const api = useApi()

const { data: stats, refresh: refreshStats } = useAsyncData('server-stats', () => api.getServerStats(), { default: () => null })
const { data: projects, pending: projectsPending } = useAsyncData('projects-overview', () => api.getProjects(), { default: () => [] })
const { data: recentDeployments } = useAsyncData('recent-deployments', () => api.getRecentDeployments(), { default: () => [] })

const runningProjects = computed(() => (projects.value as Project[]).filter(p => p.latest_deployment?.status === 'running'))
const failedProjects = computed(() => (projects.value as Project[]).filter(p => p.latest_deployment?.status === 'failed'))

useIntervalFn(refreshStats, 3000)

// Theme-aware SVG colors
const colorMode = useColorMode()
const isDark = computed(() => colorMode.value === 'dark')
const svgTrack = computed(() => isDark.value ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.10)')
const svgTextDim = computed(() => isDark.value ? 'rgba(156,163,175,0.75)' : 'rgba(75,85,99,0.85)')
const svgScaleText = computed(() => isDark.value ? 'rgba(255,255,255,0.28)' : 'rgba(0,0,0,0.35)')

// Semicircle gauge
const SEMI_R = 85
const SEMI_ARC = Math.PI * SEMI_R
function arcDash(pct: number) {
  const c = Math.min(100, Math.max(0, pct))
  return `${(SEMI_ARC * c / 100).toFixed(2)} ${(SEMI_ARC + 2).toFixed(2)}`
}

// Disk ring
const RING_R = 28
const RING_CIRC = 2 * Math.PI * RING_R

const ramPct = computed(() => {
  if (!stats.value?.memory_total) return 0
  return (stats.value.memory_used / stats.value.memory_total) * 100
})

const diskPct = computed(() => {
  if (!stats.value?.disk_total) return 0
  return (stats.value.disk_used / stats.value.disk_total) * 100
})

const cpuColor = computed(() => {
  if (!stats.value) return '#8b5cf6'
  return stats.value.cpu_usage > 80 ? '#ef4444' : stats.value.cpu_usage > 60 ? '#f59e0b' : '#8b5cf6'
})
const ramColor = computed(() => ramPct.value > 80 ? '#ef4444' : ramPct.value > 60 ? '#f59e0b' : '#06b6d4')
const diskColor = computed(() => diskPct.value > 80 ? '#ef4444' : '#f59e0b')

function formatBytes(bytes: number) {
  if (!bytes) return '0 GB'
  return (bytes / 1024 / 1024 / 1024).toFixed(1) + ' GB'
}

function formatDepDuration(started: string, finished: string) {
  const s = Math.round((new Date(finished).getTime() - new Date(started).getTime()) / 1000)
  const m = Math.floor(s / 60)
  if (m > 0) return `${m}m ${s % 60}s`
  return `${s}s`
}

function formatUptime(seconds: number) {
  if (!seconds) return '—'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  return `${h}h ${m}m`
}

const statusBadgeColor = (status?: string) => {
  if (status === 'running') return 'success'
  if (status === 'failed') return 'error'
  if (status === 'building') return 'warning'
  return 'neutral'
}
</script>

<template>
  <UDashboardPanel id="overview">
    <template #header>
      <UDashboardNavbar title="Overview">
        <template #leading>
          <UDashboardSidebarCollapse class="!text-muted hover:!text-highlighted" />
        </template>
        <template #right>
          <UBadge color="success" variant="subtle" class="font-mono text-xs flex items-center gap-1.5">
            <span class="size-1.5 rounded-full bg-green-500 inline-block animate-pulse" />
            {{ stats?.running_containers ?? 0 }} running
          </UBadge>
          <UButton icon="i-lucide-plus" label="New Project" to="/projects?new=1" size="sm" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="p-6 space-y-6">

        <!-- ── Stat cards ── -->
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <UCard>
            <div class="flex items-center gap-3">
              <div class="p-2.5 rounded-xl bg-green-500/10">
                <UIcon name="i-lucide-circle-play" class="size-5 text-green-400" />
              </div>
              <div>
                <p class="text-xs text-muted font-medium uppercase tracking-wide">Running</p>
                <p class="text-2xl font-bold text-highlighted">{{ runningProjects.length }}</p>
                <p class="text-xs text-muted">apps</p>
              </div>
            </div>
          </UCard>

          <UCard>
            <div class="flex items-center gap-3">
              <div class="p-2.5 rounded-xl bg-primary/10">
                <UIcon name="i-lucide-layers" class="size-5 text-primary" />
              </div>
              <div>
                <p class="text-xs text-muted font-medium uppercase tracking-wide">Projects</p>
                <p class="text-2xl font-bold text-highlighted">{{ (projects as Project[]).length }}</p>
                <p class="text-xs text-muted">total</p>
              </div>
            </div>
          </UCard>

          <UCard>
            <div class="flex items-center gap-3">
              <div class="p-2.5 rounded-xl" :class="failedProjects.length > 0 ? 'bg-red-500/10' : 'bg-elevated'">
                <UIcon name="i-lucide-circle-x" class="size-5" :class="failedProjects.length > 0 ? 'text-red-400' : 'text-muted'" />
              </div>
              <div>
                <p class="text-xs text-muted font-medium uppercase tracking-wide">Failed</p>
                <p class="text-2xl font-bold" :class="failedProjects.length > 0 ? 'text-red-400' : 'text-highlighted'">
                  {{ failedProjects.length }}
                </p>
                <p class="text-xs text-muted">deployments</p>
              </div>
            </div>
          </UCard>

          <UCard>
            <div class="flex items-center gap-3">
              <div class="p-2.5 rounded-xl bg-cyan-500/10">
                <UIcon name="i-lucide-timer" class="size-5 text-cyan-400" />
              </div>
              <div>
                <p class="text-xs text-muted font-medium uppercase tracking-wide">Uptime</p>
                <p class="text-2xl font-bold text-highlighted">{{ stats ? formatUptime(stats.uptime_seconds) : '—' }}</p>
                <p class="text-xs text-muted">server</p>
              </div>
            </div>
          </UCard>
        </div>

        <!-- ── CPU + RAM gauges ── -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">

          <!-- CPU Gauge -->
          <UCard>
            <template #header>
              <div class="flex items-center gap-2">
                <div class="p-1.5 rounded-lg bg-violet-500/10">
                  <UIcon name="i-lucide-cpu" class="size-4 text-violet-400" />
                </div>
                <h3 class="text-sm font-semibold">CPU Usage</h3>
              </div>
            </template>

            <div class="flex flex-col items-center py-1">
              <svg viewBox="0 0 200 120" class="w-full max-w-[220px]">
                <!-- Background track -->
                <path d="M 15 105 A 85 85 0 0 1 185 105" fill="none" :stroke="svgTrack" stroke-width="14" stroke-linecap="round" />
                <!-- Threshold marker 80% -->
                <path d="M 15 105 A 85 85 0 0 1 185 105" fill="none" stroke="rgba(239,68,68,0.22)" stroke-width="14" stroke-linecap="round"
                  :stroke-dasharray="`${(SEMI_ARC * 0.2).toFixed(2)} ${(SEMI_ARC + 2).toFixed(2)}`"
                  :stroke-dashoffset="`-${(SEMI_ARC * 0.8).toFixed(2)}`"
                />
                <!-- Value arc -->
                <path
                  d="M 15 105 A 85 85 0 0 1 185 105"
                  fill="none"
                  :stroke="cpuColor"
                  stroke-width="14"
                  stroke-linecap="round"
                  :stroke-dasharray="arcDash(stats?.cpu_usage ?? 0)"
                  style="transition: stroke-dasharray 0.6s ease, stroke 0.3s ease"
                />
                <!-- Center value -->
                <text x="100" y="86" text-anchor="middle" font-size="32" font-weight="700" font-family="ui-monospace,monospace" :fill="cpuColor" style="transition:fill 0.3s ease">
                  {{ stats ? stats.cpu_usage.toFixed(1) : '—' }}
                </text>
                <text x="100" y="102" text-anchor="middle" font-size="13" :fill="svgTextDim">percent</text>
                <!-- Scale -->
                <text x="13" y="118" text-anchor="middle" font-size="9" :fill="svgScaleText">0%</text>
                <text x="187" y="118" text-anchor="middle" font-size="9" :fill="svgScaleText">100%</text>
              </svg>
            </div>
          </UCard>

          <!-- RAM Gauge -->
          <UCard>
            <template #header>
              <div class="flex items-center gap-2">
                <div class="p-1.5 rounded-lg bg-cyan-500/10">
                  <UIcon name="i-lucide-memory-stick" class="size-4 text-cyan-400" />
                </div>
                <h3 class="text-sm font-semibold">Memory Usage</h3>
              </div>
            </template>

            <div class="flex flex-col items-center py-1">
              <svg viewBox="0 0 200 120" class="w-full max-w-[220px]">
                <!-- Background track -->
                <path d="M 15 105 A 85 85 0 0 1 185 105" fill="none" :stroke="svgTrack" stroke-width="14" stroke-linecap="round" />
                <!-- Threshold marker 80% -->
                <path d="M 15 105 A 85 85 0 0 1 185 105" fill="none" stroke="rgba(239,68,68,0.22)" stroke-width="14" stroke-linecap="round"
                  :stroke-dasharray="`${(SEMI_ARC * 0.2).toFixed(2)} ${(SEMI_ARC + 2).toFixed(2)}`"
                  :stroke-dashoffset="`-${(SEMI_ARC * 0.8).toFixed(2)}`"
                />
                <!-- Value arc -->
                <path
                  d="M 15 105 A 85 85 0 0 1 185 105"
                  fill="none"
                  :stroke="ramColor"
                  stroke-width="14"
                  stroke-linecap="round"
                  :stroke-dasharray="arcDash(ramPct)"
                  style="transition: stroke-dasharray 0.6s ease, stroke 0.3s ease"
                />
                <!-- Used / Total -->
                <text x="100" y="80" text-anchor="middle" font-size="22" font-weight="700" font-family="ui-monospace,monospace" :fill="ramColor" style="transition:fill 0.3s ease">
                  {{ stats ? formatBytes(stats.memory_used) : '—' }}
                </text>
                <text x="100" y="96" text-anchor="middle" font-size="12" :fill="svgTextDim">
                  / {{ stats ? formatBytes(stats.memory_total) : '—' }}
                </text>
                <text x="100" y="109" text-anchor="middle" font-size="11" :fill="ramColor" style="transition:fill 0.3s ease">
                  {{ ramPct.toFixed(1) }}%
                </text>
                <!-- Scale -->
                <text x="13" y="118" text-anchor="middle" font-size="9" :fill="svgScaleText">0%</text>
                <text x="187" y="118" text-anchor="middle" font-size="9" :fill="svgScaleText">100%</text>
              </svg>
            </div>
          </UCard>
        </div>

        <!-- ── Disk + Recent Projects ── -->
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">

          <!-- Disk ring -->
          <UCard>
            <template #header>
              <div class="flex items-center gap-2">
                <div class="p-1.5 rounded-lg bg-amber-500/10">
                  <UIcon name="i-lucide-hard-drive" class="size-4 text-amber-400" />
                </div>
                <h3 class="text-sm font-semibold">Disk Usage</h3>
              </div>
            </template>

            <div class="flex items-center justify-center gap-6 py-2">
              <svg width="80" height="80" viewBox="0 0 80 80" class="shrink-0 -rotate-90">
                <circle cx="40" cy="40" :r="RING_R" fill="none" :stroke="svgTrack" stroke-width="7" />
                <circle
                  cx="40" cy="40" :r="RING_R" fill="none"
                  :stroke="diskColor"
                  stroke-width="7"
                  stroke-linecap="round"
                  :stroke-dasharray="RING_CIRC"
                  :stroke-dashoffset="RING_CIRC * (1 - diskPct / 100)"
                  style="transition: stroke-dashoffset 0.6s ease"
                />
              </svg>
              <div>
                <p class="text-3xl font-bold font-mono" :style="{ color: diskColor }">
                  {{ diskPct.toFixed(0) }}<span class="text-lg">%</span>
                </p>
                <p class="text-sm text-muted mt-1">{{ stats ? formatBytes(stats.disk_used) : '—' }} used</p>
                <p class="text-xs text-muted">of {{ stats ? formatBytes(stats.disk_total) : '—' }}</p>
              </div>
            </div>
          </UCard>

          <!-- Recent Projects -->
          <UCard class="lg:col-span-2">
            <template #header>
              <div class="flex items-center justify-between">
                <h3 class="text-sm font-semibold">Recent Projects</h3>
                <UButton variant="ghost" size="xs" to="/projects" trailing-icon="i-lucide-arrow-right" color="neutral">
                  View all
                </UButton>
              </div>
            </template>

            <div v-if="projectsPending" class="flex justify-center py-8">
              <UIcon name="i-lucide-loader-circle" class="size-6 text-muted animate-spin" />
            </div>
            <div v-else-if="!(projects as Project[]).length" class="py-8 text-center text-muted text-sm">
              No projects yet.
              <NuxtLink to="/projects?new=1" class="text-primary hover:underline ml-1">Deploy your first app →</NuxtLink>
            </div>
            <div v-else class="divide-y divide-default">
              <div
                v-for="project in [...(projects as Project[])].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()).slice(0, 4)"
                :key="project.id"
                class="flex items-center justify-between py-3 gap-3 group"
              >
                <div class="flex items-center gap-3 min-w-0">
                  <div class="relative shrink-0">
                    <div class="p-1.5 rounded-lg bg-primary/10 border border-primary/20">
                      <UIcon name="i-lucide-box" class="size-4 text-primary" />
                    </div>
                    <span
                      class="absolute -bottom-0.5 -right-0.5 size-2 rounded-full border border-default"
                      :class="{
                        'bg-green-500': project.latest_deployment?.status === 'running',
                        'bg-red-500': project.latest_deployment?.status === 'failed',
                        'bg-yellow-500 animate-pulse': project.latest_deployment?.status === 'building',
                        'bg-zinc-500': !project.latest_deployment || project.latest_deployment?.status === 'stopped'
                      }"
                    />
                  </div>
                  <div class="min-w-0">
                    <p class="text-sm font-medium text-highlighted truncate">{{ project.name }}</p>
                    <p class="text-xs text-muted truncate font-mono">{{ project.domain }}</p>
                  </div>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <UBadge :color="statusBadgeColor(project.latest_deployment?.status)" variant="subtle" size="sm">
                    {{ project.latest_deployment?.status ?? 'not deployed' }}
                  </UBadge>
                  <UButton
                    :to="`/projects/${project.id}`"
                    variant="ghost" size="xs" icon="i-lucide-arrow-right" color="neutral"
                  />
                </div>
              </div>
            </div>
          </UCard>
        </div>

        <!-- ── Failed alert (above activity) ── -->
        <UAlert
          v-if="failedProjects.length > 0"
          color="error"
          variant="subtle"
          :title="`${failedProjects.length} project${failedProjects.length > 1 ? 's' : ''} failed`"
          :description="failedProjects.map(p => p.name).join(', ')"
          icon="i-lucide-triangle-alert"
        />

        <!-- ── Deployment Activity ── -->
        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <div class="p-1.5 rounded-lg bg-violet-500/10">
                  <UIcon name="i-lucide-activity" class="size-4 text-violet-400" />
                </div>
                <h3 class="text-sm font-semibold">Deployment Activity</h3>
              </div>
              <UButton variant="ghost" size="xs" to="/projects" trailing-icon="i-lucide-arrow-right" color="neutral">All projects</UButton>
            </div>
          </template>

          <div v-if="!(recentDeployments as RecentDeployment[]).length" class="py-6 text-center text-sm text-muted">
            No deployments yet
          </div>
          <div v-else class="divide-y divide-default">
            <div
              v-for="dep in (recentDeployments as RecentDeployment[]).slice(0, 5)"
              :key="dep.id"
              class="flex items-center justify-between py-3 gap-3"
            >
              <div class="flex items-center gap-3 min-w-0">
                <UBadge :color="statusBadgeColor(dep.status)" variant="subtle" size="sm">{{ dep.status }}</UBadge>
                <div class="min-w-0">
                  <NuxtLink :to="`/projects/${dep.project_id}`" class="text-sm font-medium text-highlighted hover:text-primary transition-colors truncate">
                    {{ dep.project_name }}
                  </NuxtLink>
                  <p class="text-xs text-muted">
                    <span v-if="dep.commit_sha" class="font-mono mr-1.5 text-primary">{{ dep.commit_sha.slice(0, 7) }}</span>
                    {{ new Date(dep.started_at).toLocaleString() }}
                  </p>
                </div>
              </div>
              <div class="shrink-0 text-xs text-muted font-mono">
                {{ dep.finished_at ? formatDepDuration(dep.started_at, dep.finished_at) : '…' }}
              </div>
            </div>
          </div>
        </UCard>


      </div>
    </template>
  </UDashboardPanel>
</template>
