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
	repo_url: string;
	status: 'active' | 'deploying' | 'valid' | 'error';
	serviceCount: number;
	createdAt: string;
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
