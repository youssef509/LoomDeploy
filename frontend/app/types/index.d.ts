export type UserRole = 'admin' | 'developer' | 'viewer'
export type DeploymentStatus = 'pending' | 'building' | 'running' | 'failed' | 'stopped'
export type SourceType = 'git' | 'upload'

export interface AuthUser {
  id: string
  email: string
  role: UserRole
  created_at: string
}

export interface ProjectDomain {
  id: string
  project_id: string
  domain: string
  created_at: string
}

export interface DomainList {
  primary_domain: string
  is_generated: boolean
  extra_domains: ProjectDomain[]
}

export interface Project {
  id: string
  user_id: string
  name: string
  source_type: SourceType
  repository_url?: string
  branch?: string
  domain: string
  is_generated_domain?: boolean
  extra_domains?: ProjectDomain[]
  container_port: number
  has_git_token?: boolean
  webhook_secret?: string
  has_webhook?: boolean
  notification_webhook_url?: string
  cpu_limit?: number
  memory_limit_mb?: number
  health_check_url?: string
  dockerfile_content?: string
  created_at: string
  latest_deployment?: Deployment
}

export interface GitHubAppStatus {
  phase: 'unconfigured' | 'setup' | 'connected'
  app_name?: string
  app_id?: string
  installation_id?: number
}

export interface GitHubRepo {
  full_name: string
  clone_url: string
  default_branch: string
  private: boolean
}

export interface RecentDeployment {
  id: string
  project_id: string
  project_name: string
  status: DeploymentStatus
  commit_sha?: string
  started_at: string
  finished_at?: string
}

export interface Deployment {
  id: string
  project_id: string
  status: DeploymentStatus
  container_id?: string
  commit_sha?: string
  build_logs?: string
  started_at: string
  finished_at?: string
}

export interface EnvVar {
  id: string
  project_id: string
  key: string
  value: string
}

export interface ContainerStats {
  cpu_percent: number
  memory_usage_mb: number
  memory_limit_mb: number
  memory_percent: number
}

export interface SourceToken {
  provider: string
  label: string
  has_token: boolean
  updated_at: string
}

export interface ServerStats {
  cpu_usage: number
  memory_used: number
  memory_total: number
  disk_used: number
  disk_total: number
  uptime_seconds: number
  running_containers: number
  total_containers: number
}
