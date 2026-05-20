import type { Project, Deployment, EnvVar, ServerStats, SourceToken, ContainerStats, RecentDeployment, GitHubAppStatus, GitHubRepo, DomainList } from '~/types'

export const useApi = () => {
  const auth = useAuthStore()

  const headers = computed(() => ({
    Authorization: `Bearer ${auth.token ?? ''}`
  }))

  function apiFetch<T>(path: string, options?: Parameters<typeof $fetch>[1]) {
    return $fetch<T>(path, {
      ...options,
      headers: {
        ...headers.value,
        ...(options?.headers ?? {})
      }
    })
  }

  const getSetupStatus = () => apiFetch<{ needs_setup: boolean }>('/api/auth/setup-status')
  const setupAdmin = (email: string, password: string) =>
    apiFetch<{ message: string, id: string }>('/api/auth/register', { method: 'POST', body: { email, password } })

  const getServerStats = () => apiFetch<ServerStats>('/api/system/stats')

  const getProjects = () => apiFetch<Project[]>('/api/projects')
  const getProject = (id: string) => apiFetch<Project>(`/api/projects/${id}`)
  const createProject = (body: Partial<Project> & { git_token?: string }) =>
    apiFetch<Project>('/api/projects', { method: 'POST', body })
  const updateProject = (id: string, body: Partial<Pick<Project, 'name' | 'domain' | 'repository_url' | 'branch' | 'container_port'>> & { git_token?: string }) =>
    apiFetch<Project>(`/api/projects/${id}`, { method: 'PATCH', body })
  const deleteProject = (id: string) => apiFetch<void>(`/api/projects/${id}`, { method: 'DELETE' })

  const deployProject = (id: string) => apiFetch<Deployment>(`/api/projects/${id}/deploy`, { method: 'POST' })
  const getDeployments = (projectId: string) => apiFetch<Deployment[]>(`/api/projects/${projectId}/deployments`)
  const getRecentDeployments = () => apiFetch<RecentDeployment[]>('/api/deployments/recent')
  const deleteDeployment = (projectId: string, depId: string) =>
    apiFetch<void>(`/api/projects/${projectId}/deployments/${depId}`, { method: 'DELETE' })

  const changePassword = (current_password: string, new_password: string) =>
    apiFetch<{ message: string }>('/api/auth/change-password', { method: 'POST', body: { current_password, new_password } })

  const getGitHubAppStatus = () => apiFetch<GitHubAppStatus>('/api/github/app/status')
  const getGitHubManifest = () => apiFetch<Record<string, unknown>>('/api/github/app/manifest')
  const getGitHubRepos = () => apiFetch<GitHubRepo[]>('/api/github/app/repos')
  const disconnectGitHubApp = () => apiFetch<void>('/api/github/app/disconnect', { method: 'DELETE' })
  const resetGitHubApp = () => apiFetch<void>('/api/github/app/reset', { method: 'DELETE' })

  const containerAction = (projectId: string, action: 'start' | 'stop' | 'restart') =>
    apiFetch<void>(`/api/projects/${projectId}/container/${action}`, { method: 'POST' })

  const getEnvVars = (projectId: string) => apiFetch<EnvVar[]>(`/api/projects/${projectId}/env`)
  const setEnvVars = (projectId: string, vars: { key: string, value: string }[]) =>
    apiFetch<EnvVar[]>(`/api/projects/${projectId}/env`, { method: 'PUT', body: vars })

  const redeployDeployment = (projectId: string, depId: string) =>
    apiFetch<Deployment>(`/api/projects/${projectId}/deployments/${depId}/redeploy`, { method: 'POST' })

  const getContainerStats = (id: string) =>
    apiFetch<ContainerStats | { message: string }>(`/api/projects/${id}/stats`)

  const regenerateWebhook = (id: string) =>
    apiFetch<{ webhook_secret: string }>(`/api/projects/${id}/webhook/regenerate`, { method: 'POST' })

  const getSourceTokens = () => apiFetch<SourceToken[]>('/api/settings/source-control')
  const setSourceToken = (provider: string, body: { token: string, label?: string }) =>
    apiFetch<SourceToken>(`/api/settings/source-control/${provider}`, { method: 'PUT', body })
  const deleteSourceToken = (provider: string) =>
    apiFetch<void>(`/api/settings/source-control/${provider}`, { method: 'DELETE' })

  const listDomains = (projectId: string) =>
    apiFetch<DomainList>(`/api/projects/${projectId}/domains`)
  const addDomain = (projectId: string, domain: string) =>
    apiFetch<{ id: string, domain: string }>(`/api/projects/${projectId}/domains`, { method: 'POST', body: { domain } })
  const removeDomain = (projectId: string, domainId: string) =>
    apiFetch<void>(`/api/projects/${projectId}/domains/${domainId}`, { method: 'DELETE' })
  const updatePrimaryDomain = (projectId: string, domain: string) =>
    apiFetch<{ domain: string }>(`/api/projects/${projectId}/domains/primary`, { method: 'PUT', body: { domain } })

  return {
    getSetupStatus,
    setupAdmin,
    getServerStats,
    getProjects,
    getProject,
    createProject,
    updateProject,
    deleteProject,
    deployProject,
    getDeployments,
    containerAction,
    getEnvVars,
    setEnvVars,
    getRecentDeployments,
    deleteDeployment,
    changePassword,
    getGitHubAppStatus,
    getGitHubManifest,
    getGitHubRepos,
    disconnectGitHubApp,
    resetGitHubApp,
    redeployDeployment,
    getContainerStats,
    regenerateWebhook,
    getSourceTokens,
    setSourceToken,
    deleteSourceToken,
    listDomains,
    addDomain,
    removeDomain,
    updatePrimaryDomain
  }
}
