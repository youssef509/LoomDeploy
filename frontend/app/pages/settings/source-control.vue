<script setup lang="ts">
import type { SourceToken, GitHubAppStatus, GitHubRepo } from '~/types'

definePageMeta({ middleware: 'auth' })

const api = useApi()
const toast = useToast()
const route = useRoute()

// ── GitHub App ───────────────────────────────────────────────────────────────
const { data: ghStatus, refresh: refreshGh } = await useAsyncData(
  'gh-app-status-sc',
  () => api.getGitHubAppStatus(),
  { default: () => ({ phase: 'unconfigured' }) as GitHubAppStatus }
)

const ghRepos = ref<GitHubRepo[]>([])
const ghReposLoading = ref(false)
const registering = ref(false)
const disconnecting = ref(false)
const resetting = ref(false)

// Show toast if GitHub redirected back with a result
onMounted(async () => {
  if (route.query.gh_connected) {
    toast.add({ title: 'GitHub connected!', description: 'Your repositories are now accessible.', color: 'success' })
    await refreshGh()
  }
  if (route.query.gh_error) {
    toast.add({ title: 'GitHub connection failed', description: String(route.query.gh_error), color: 'error' })
  }
  if ((ghStatus.value as GitHubAppStatus).phase === 'connected') {
    loadRepos()
  }
})

async function loadRepos() {
  ghReposLoading.value = true
  try { ghRepos.value = await api.getGitHubRepos() }
  catch { ghRepos.value = [] }
  finally { ghReposLoading.value = false }
}

async function registerApp() {
  registering.value = true
  try {
    const manifest = await api.getGitHubManifest()
    const form = document.createElement('form')
    form.method = 'POST'
    form.action = 'https://github.com/settings/apps/new'
    form.target = '_self'
    const input = document.createElement('input')
    input.type = 'hidden'
    input.name = 'manifest'
    input.value = JSON.stringify(manifest)
    form.appendChild(input)
    document.body.appendChild(form)
    form.submit()
  } catch {
    toast.add({ title: 'Failed to start GitHub App registration', color: 'error' })
    registering.value = false
  }
}

async function installApp() {
  const name = (ghStatus.value as GitHubAppStatus).app_name
  if (name) window.location.href = `https://github.com/apps/${name}/installations/new`
}

async function disconnect() {
  disconnecting.value = true
  try {
    await api.disconnectGitHubApp()
    toast.add({ title: 'GitHub installation removed', color: 'success' })
    ghRepos.value = []
    refreshGh()
  } catch {
    toast.add({ title: 'Failed to disconnect', color: 'error' })
  } finally {
    disconnecting.value = false
  }
}

async function resetApp() {
  resetting.value = true
  try {
    await api.resetGitHubApp()
    toast.add({ title: 'GitHub App removed', color: 'success' })
    ghRepos.value = []
    refreshGh()
  } catch {
    toast.add({ title: 'Failed to reset', color: 'error' })
  } finally {
    resetting.value = false
  }
}

// ── PAT providers (GitLab, Bitbucket, custom) ────────────────────────────────
const { data: tokens, refresh: refreshTokens } = await useAsyncData('source-tokens', () => api.getSourceTokens())

const PAT_PROVIDERS = [
  {
    id: 'gitlab.com',
    label: 'GitLab',
    icon: 'i-simple-icons-gitlab',
    description: 'Clone from GitLab repositories (public & private)',
    patUrl: 'https://gitlab.com/-/profile/personal_access_tokens',
    placeholder: 'glpat-xxxxxxxxxxxxxxxxxxxx'
  },
  {
    id: 'bitbucket.org',
    label: 'Bitbucket',
    icon: 'i-simple-icons-bitbucket',
    description: 'Clone from Bitbucket repositories (public & private)',
    patUrl: 'https://bitbucket.org/account/settings/app-passwords/new',
    placeholder: 'ATBBxxxxxxxxxxxxxxxxxxxxxxxx'
  }
]

const tokenMap = computed(() => {
  const map: Record<string, SourceToken> = {}
  for (const t of tokens.value ?? []) map[t.provider] = t
  return map
})

const forms = reactive<Record<string, { token: string, open: boolean, saving: boolean }>>({})
function getForm(provider: string) {
  if (!forms[provider]) forms[provider] = { token: '', open: false, saving: false }
  return forms[provider]
}

async function saveToken(provider: string, label: string) {
  const form = getForm(provider)
  if (!form.token.trim()) return
  form.saving = true
  try {
    await api.setSourceToken(provider, { token: form.token.trim(), label })
    toast.add({ title: `${label} connected`, color: 'success' })
    form.token = ''; form.open = false
    refreshTokens()
  } catch { toast.add({ title: 'Failed to save token', color: 'error' }) }
  finally { form.saving = false }
}

async function removeToken(provider: string, label: string) {
  try {
    await api.deleteSourceToken(provider)
    toast.add({ title: `${label} disconnected`, color: 'success' })
    refreshTokens()
  } catch { toast.add({ title: 'Failed to remove token', color: 'error' }) }
}

const customForm = reactive({ host: '', token: '', saving: false })
async function saveCustomToken() {
  if (!customForm.host.trim() || !customForm.token.trim()) return
  customForm.saving = true
  try {
    const host = customForm.host.trim().replace(/^https?:\/\//, '').replace(/\/$/, '')
    await api.setSourceToken(host, { token: customForm.token.trim(), label: host })
    toast.add({ title: `${host} connected`, color: 'success' })
    customForm.host = ''; customForm.token = ''
    refreshTokens()
  } catch { toast.add({ title: 'Failed to save token', color: 'error' }) }
  finally { customForm.saving = false }
}

const customTokens = computed(() =>
  (tokens.value ?? []).filter(t => !PAT_PROVIDERS.find(p => p.id === t.provider) && t.provider !== 'github_app')
)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-base font-semibold">Source Control</h2>
      <p class="text-sm text-muted mt-1">Connect your Git providers to deploy private repositories.</p>
    </div>

    <!-- ── GitHub App card ─────────────────────────────────────────────── -->
    <UCard>
      <div class="flex items-center gap-3 mb-5">
        <div class="size-10 rounded-xl bg-elevated border border-default flex items-center justify-center shrink-0">
          <UIcon name="i-simple-icons-github" class="size-5 text-highlighted" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <span class="font-semibold text-sm">GitHub</span>
            <UBadge
              v-if="(ghStatus as GitHubAppStatus).phase === 'connected'"
              color="success" variant="subtle" size="xs" icon="i-lucide-check" label="Connected"
            />
            <UBadge
              v-else-if="(ghStatus as GitHubAppStatus).phase === 'setup'"
              color="warning" variant="subtle" size="xs" icon="i-lucide-clock" label="Pending install"
            />
            <UBadge v-else color="neutral" variant="subtle" size="xs" label="Not connected" />
          </div>
          <p class="text-xs text-muted mt-0.5">Deploy private GitHub repos — no token copy-pasting needed</p>
        </div>
      </div>

      <!-- Phase 1: Unconfigured — register the app -->
      <div v-if="(ghStatus as GitHubAppStatus).phase === 'unconfigured'" class="space-y-4">
        <div class="rounded-xl border border-default bg-elevated/40 p-4 space-y-3">
          <p class="text-sm font-medium text-highlighted">How it works</p>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div class="flex items-start gap-2.5">
              <span class="flex items-center justify-center size-6 rounded-full bg-primary/15 text-primary text-xs font-bold shrink-0 mt-0.5">1</span>
              <span class="text-xs text-muted">Click <strong class="text-highlighted">Register GitHub App</strong> — LoomDeploy creates a GitHub App on your account automatically.</span>
            </div>
            <div class="flex items-start gap-2.5">
              <span class="flex items-center justify-center size-6 rounded-full bg-primary/15 text-primary text-xs font-bold shrink-0 mt-0.5">2</span>
              <span class="text-xs text-muted">GitHub redirects you to <strong class="text-highlighted">install</strong> the app and choose which repos to grant access to.</span>
            </div>
            <div class="flex items-start gap-2.5">
              <span class="flex items-center justify-center size-6 rounded-full bg-primary/15 text-primary text-xs font-bold shrink-0 mt-0.5">3</span>
              <span class="text-xs text-muted">Done. <strong class="text-highlighted">Pick any repo</strong> in project settings — LoomDeploy handles tokens automatically.</span>
            </div>
          </div>
          <div class="flex flex-wrap gap-2 pt-1">
            <span class="inline-flex items-center gap-1 text-xs text-green-500"><UIcon name="i-lucide-check-circle" class="size-3.5" /> No PAT needed</span>
            <span class="inline-flex items-center gap-1 text-xs text-green-500"><UIcon name="i-lucide-check-circle" class="size-3.5" /> Works with private repos</span>
            <span class="inline-flex items-center gap-1 text-xs text-green-500"><UIcon name="i-lucide-check-circle" class="size-3.5" /> Tokens auto-renewed</span>
          </div>
        </div>
        <div class="flex justify-end">
          <UButton icon="i-simple-icons-github" :loading="registering" @click="registerApp">
            Register GitHub App
          </UButton>
        </div>
      </div>

      <!-- Phase 2: App created, awaiting installation -->
      <div v-else-if="(ghStatus as GitHubAppStatus).phase === 'setup'" class="space-y-4">
        <div class="rounded-xl border border-warning/30 bg-warning/5 p-4 flex items-start gap-3">
          <UIcon name="i-lucide-info" class="size-4 text-warning shrink-0 mt-0.5" />
          <div class="text-sm">
            <p class="font-medium text-highlighted">GitHub App created — one more step!</p>
            <p class="text-muted text-xs mt-1">
              Your app <code class="font-mono bg-default px-1 rounded">{{ (ghStatus as GitHubAppStatus).app_name }}</code>
              was registered. Now install it on your GitHub account to grant repo access.
            </p>
          </div>
        </div>
        <div class="flex items-center justify-between">
          <UButton variant="ghost" color="error" size="sm" icon="i-lucide-rotate-ccw" :loading="resetting" @click="resetApp">
            Start over
          </UButton>
          <UButton icon="i-lucide-plug" @click="installApp">
            Install on GitHub Account
          </UButton>
        </div>
      </div>

      <!-- Phase 3: Fully connected -->
      <div v-else class="space-y-4">
        <div class="flex items-center justify-between p-3 rounded-xl bg-green-500/5 border border-green-500/20">
          <div class="flex items-center gap-3">
            <UIcon name="i-lucide-check-circle" class="size-4 text-green-500 shrink-0" />
            <div>
              <p class="text-sm font-medium text-highlighted">{{ (ghStatus as GitHubAppStatus).app_name }}</p>
              <p class="text-xs text-muted font-mono">Installation #{{ (ghStatus as GitHubAppStatus).installation_id }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <UButton variant="ghost" size="xs" icon="i-lucide-settings" color="neutral" @click="installApp">
              Manage on GitHub
            </UButton>
            <UButton variant="ghost" size="xs" icon="i-lucide-unplug" color="error" :loading="disconnecting" @click="disconnect">
              Disconnect
            </UButton>
          </div>
        </div>

        <!-- Repo list -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <p class="text-xs font-medium text-muted uppercase tracking-wide">Accessible repositories</p>
            <UButton variant="ghost" size="xs" icon="i-lucide-refresh-cw" color="neutral" :loading="ghReposLoading" @click="loadRepos">
              Refresh
            </UButton>
          </div>
          <div v-if="ghReposLoading" class="flex items-center gap-2 text-sm text-muted py-4">
            <UIcon name="i-lucide-loader-circle" class="size-4 animate-spin" /> Loading repositories…
          </div>
          <div v-else-if="ghRepos.length === 0" class="py-4 text-center text-xs text-muted">
            No repositories found. Make sure you granted access when installing the app.
          </div>
          <div v-else class="rounded-xl border border-default divide-y divide-default max-h-64 overflow-y-auto">
            <div
              v-for="repo in ghRepos" :key="repo.full_name"
              class="flex items-center justify-between px-3 py-2 text-sm"
            >
              <span class="flex items-center gap-2 min-w-0">
                <UIcon :name="repo.private ? 'i-lucide-lock' : 'i-lucide-book-open'" class="size-3.5 text-muted shrink-0" />
                <span class="font-mono text-xs truncate">{{ repo.full_name }}</span>
              </span>
              <UBadge size="xs" variant="subtle" color="neutral" class="shrink-0 ml-2">{{ repo.default_branch }}</UBadge>
            </div>
          </div>
          <p class="text-xs text-muted mt-2">
            {{ ghRepos.length }} repo{{ ghRepos.length !== 1 ? 's' : '' }} accessible ·
            <a class="text-primary cursor-pointer underline" @click="installApp">manage on GitHub</a> to add more
          </p>
        </div>
      </div>
    </UCard>

    <!-- ── GitLab & Bitbucket PAT cards ────────────────────────────────── -->
    <div class="space-y-3">
      <UCard v-for="p in PAT_PROVIDERS" :key="p.id">
        <div class="flex items-start justify-between gap-4">
          <div class="flex items-center gap-3 min-w-0">
            <div class="flex items-center justify-center size-10 rounded-lg bg-elevated shrink-0">
              <UIcon :name="p.icon" class="size-5" />
            </div>
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-medium text-sm">{{ p.label }}</span>
                <UBadge v-if="tokenMap[p.id]?.has_token" color="success" variant="subtle" size="xs" icon="i-lucide-check" label="Connected" />
                <UBadge v-else color="neutral" variant="subtle" size="xs" label="Not connected" />
              </div>
              <p class="text-xs text-muted mt-0.5">{{ p.description }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <UButton v-if="tokenMap[p.id]?.has_token" size="xs" color="error" variant="ghost" icon="i-lucide-trash-2" @click="removeToken(p.id, p.label)" />
            <UButton
              size="xs"
              :variant="tokenMap[p.id]?.has_token ? 'outline' : 'solid'"
              :icon="tokenMap[p.id]?.has_token ? 'i-lucide-rotate-cw' : 'i-lucide-plus'"
              @click="getForm(p.id).open = !getForm(p.id).open"
            >{{ tokenMap[p.id]?.has_token ? 'Replace' : 'Add Token' }}</UButton>
          </div>
        </div>
        <div v-if="getForm(p.id).open" class="mt-4 pt-4 border-t border-default space-y-3">
          <div class="rounded-lg bg-elevated/60 border border-default px-4 py-3 space-y-2">
            <p class="text-xs font-semibold uppercase tracking-wide">How to connect</p>
            <ol class="space-y-1.5 text-xs text-muted">
              <li>1. <a :href="p.patUrl" target="_blank" class="text-primary underline">Open {{ p.label }} → Create a new token</a></li>
              <li>2. Select scope <code class="bg-default border border-default px-1 rounded">read_repository</code> and generate.</li>
              <li>3. Paste below and click <strong>Save</strong>.</li>
            </ol>
          </div>
          <div class="flex gap-2">
            <UInput v-model="getForm(p.id).token" type="password" :placeholder="p.placeholder" class="flex-1" @keyup.enter="saveToken(p.id, p.label)" />
            <UButton :loading="getForm(p.id).saving" :disabled="!getForm(p.id).token" @click="saveToken(p.id, p.label)">Save</UButton>
          </div>
        </div>
      </UCard>
    </div>

    <!-- ── Custom / self-hosted ────────────────────────────────────────── -->
    <UCard>
      <template #header>
        <div>
          <h3 class="text-sm font-semibold">Custom / Self-hosted Provider</h3>
          <p class="text-xs text-muted mt-0.5">Gitea, Forgejo, self-hosted GitLab, Gogs, etc.</p>
        </div>
      </template>
      <div v-if="customTokens.length" class="space-y-2 mb-4">
        <div v-for="t in customTokens" :key="t.provider" class="flex items-center justify-between px-3 py-2 rounded-lg bg-elevated text-sm">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-git-branch" class="size-4 text-muted" />
            <span class="font-mono text-xs">{{ t.provider }}</span>
            <UBadge color="success" variant="subtle" size="xs" icon="i-lucide-check" label="Connected" />
          </div>
          <UButton size="xs" color="error" variant="ghost" icon="i-lucide-trash-2" @click="removeToken(t.provider, t.provider)" />
        </div>
      </div>
      <div class="space-y-3">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <UFormField label="Git Host" hint="e.g. git.mycompany.com">
            <UInput v-model="customForm.host" placeholder="git.mycompany.com" class="w-full" />
          </UFormField>
          <UFormField label="Personal Access Token">
            <UInput v-model="customForm.token" type="password" placeholder="token" class="w-full" />
          </UFormField>
        </div>
        <div class="flex justify-end">
          <UButton icon="i-lucide-plus" :loading="customForm.saving" :disabled="!customForm.host || !customForm.token" @click="saveCustomToken">
            Add Provider
          </UButton>
        </div>
      </div>
    </UCard>
  </div>
</template>
