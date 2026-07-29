export interface Container {
	id: string;
	name: string;
	image: string;
	status: 'running' | 'stopped' | 'restarting' | 'error';
	port: string;
	cpuPercent: number;
	memoryMb: number;
	uptime: string;
}

export interface Volume {
	Name: string;
	Driver: string;
	Mountpoint: string;
	CreatedAt?: string;
	Scope?: string;
}

export interface SystemStats {
	totalContainers: number;
	activeContainers: number;
	stoppedContainers: number;
	totalMemMb: number;
	usedMemMb: number;
	memPercent: number;
	cpuPercent?: number;
	cpus: number;
	os?: string;
	arch?: string;
}

export interface BlueprintItem {
	id: string;
	name: string;
	project?: string;
	version?: string;
	repo_url?: string;
	status?: 'active' | 'deploying' | 'valid' | 'error' | 'ready' | string;
	serviceCount?: number;
	service_count_actual?: number;
	createdAt?: string;
	services?: Record<string, any>;
}

export type Blueprint = BlueprintItem;

export interface ServiceRecord {
	id?: number;
	project_id: string;
	name: string;
	type: 'web' | 'static' | 'database' | 'worker';
	env_vars: Record<string, string>;
	port: number;
	custom_domain: string;
	auto_deploy: boolean;
	build_command: string;
	start_command: string;
	instance_type: string;
	created_at?: string;
	updated_at?: string;
}

export interface DeploymentRecord {
	id: string;
	project_id: string;
	status: 'queued' | 'building' | 'live' | 'error' | 'canceled';
	commit_sha: string;
	trigger: 'manual' | 'auto' | 'rollback';
	started_at: string;
	finished_at: string;
	error: string;
}

export interface ScanService {
	name: string;
	type: 'web' | 'static' | 'database' | 'worker';
	image?: string;
	source?: {
		repo?: string;
		directory?: string;
		ref?: string;
	};
	build?: {
		engine?: string;
		command?: string;
		dockerfile_path?: string;
	};
	deploy?: {
		port?: number;
		command?: string;
		env?: Record<string, string>;
	};
	defaults: {
		env: Record<string, string>;
		port: number;
	};
}

export interface ScanResult {
	project: string;
	repo_url: string;
	services: ScanService[];
	warnings: string[];
	errors: string[];
	blueprint?: any;
}

export interface LogEvent {
	timestamp: string;
	stage: 'clone' | 'build' | 'deploy' | 'runtime' | 'system';
	service: string;
	message: string;
	level: 'info' | 'warn' | 'error' | 'success';
}

export interface ProjectDetail {
	blueprint: BlueprintItem;
	services: ServiceRecord[];
	latest?: DeploymentRecord;
}

export interface DeleteTarget {
	type: 'container' | 'volume' | 'blueprint';
	idOrName: string;
	label: string;
}

export interface ErrorModalState {
	title: string;
	message: string;
	details?: string;
}

export interface LogStreamState {
	id: string;
	name: string;
	logs: string[];
}

export type TabType = 'overview' | 'containers' | 'volumes' | 'blueprints' | 'settings';
