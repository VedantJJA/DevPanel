import React, { useState } from 'react';
import { 
  LayoutDashboard, 
  Server, 
  Database, 
  Globe, 
  Settings, 
  Terminal, 
  Activity, 
  Plus, 
  ArrowLeft, 
  Play, 
  Square,
  RefreshCw,
  Github,
  Search,
  MoreVertical,
  ChevronRight,
  ExternalLink,
  Cpu,
  Clock,
  Box,
  Layers,
  FileCode,
  HardDrive,
  Wifi,
  Trash2,
  Edit2,
  Folder,
  Menu,
  X,
  Briefcase,
  GitBranch,
  Lock,
  Unlock,
  CheckCircle,
  Eye,
  EyeOff,
  Copy,
  PlayCircle,
  StopCircle
} from 'lucide-react';

const MOCK_SERVICES = [
  {
    id: 'srv-01',
    name: 'api-gateway',
    type: 'web',
    status: 'live',
    repo: 'my-org/api-gateway',
    branch: 'main',
    url: 'api-gateway-vx8a.onrender.com',
    region: 'Frankfurt (EU)',
    lastDeploy: '10 mins ago',
    runtime: 'Node.js',
    group: 'Production'
  },
  {
    id: 'srv-03',
    name: 'primary-db',
    type: 'postgres',
    status: 'live',
    repo: '-',
    branch: '-',
    url: 'dpg-c0x8a-frankfurt-postgres.com',
    region: 'Frankfurt (EU)',
    lastDeploy: '2 days ago',
    runtime: 'PostgreSQL 15',
    group: 'Production'
  },
  {
    id: 'srv-02',
    name: 'frontend-app',
    type: 'static',
    status: 'deploying',
    repo: 'my-org/frontend',
    branch: 'staging',
    url: 'frontend-staging.onrender.com',
    region: 'Ohio (US)',
    lastDeploy: 'Just now',
    runtime: 'React',
    group: 'Staging Environment'
  },
  {
    id: 'srv-04',
    name: 'background-worker',
    type: 'cron',
    status: 'running',
    repo: 'my-org/workers',
    branch: 'main',
    url: '-',
    region: 'Ohio (US)',
    lastDeploy: '5 hrs ago',
    runtime: 'Python',
    group: 'Staging Environment'
  }
];

const MOCK_CONTAINERS = [
  { id: 'c-1', name: 'redis-cache-main', image: 'redis:7-alpine', status: 'running', ports: '6379:6379', created: '5 days ago' },
  { id: 'c-2', name: 'background-worker-1', image: 'worker-node:latest', status: 'exited', ports: '-', created: '12 hours ago' },
  { id: 'c-3', name: 'pg-admin-tools', image: 'dpage/pgadmin4', status: 'running', ports: '5050:80', created: '2 weeks ago' },
];

const MOCK_BLUEPRINTS = [
  { id: 'bp-1', name: 'LEMP Stack', description: 'Linux, Nginx, MySQL, PHP standard environment.', source: 'github.com/org/lemp-bp' },
  { id: 'bp-2', name: 'Node.js Microservice', description: 'Standardized Node.js REST API with Redis.', source: 'Local Template' },
];

const MOCK_REPOS = [
  { id: 1, name: 'my-org/frontend-app', updated: '2 hours ago', private: true },
  { id: 2, name: 'my-org/api-gateway', updated: '5 hours ago', private: true },
  { id: 3, name: 'my-org/background-workers', updated: '1 day ago', private: false },
  { id: 4, name: 'my-org/landing-page', updated: '3 days ago', private: false },
];

const StatusBadge = ({ status }) => {
  const styles = {
    live: 'bg-green-100 text-green-700 border-green-200',
    running: 'bg-green-100 text-green-700 border-green-200',
    deploying: 'bg-yellow-100 text-yellow-700 border-yellow-200 animate-pulse',
    failed: 'bg-red-100 text-red-700 border-red-200',
    exited: 'bg-gray-100 text-gray-700 border-gray-200',
    suspended: 'bg-gray-100 text-gray-700 border-gray-200',
  };

  const labels = {
    live: 'Live',
    running: 'Running',
    deploying: 'Deploying...',
    failed: 'Failed',
    exited: 'Exited',
    suspended: 'Suspended',
  };

  return (
    <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium border ${styles[status]}`}>
      {labels[status]}
    </span>
  );
};

const TypeIcon = ({ type, className = "w-5 h-5" }) => {
  switch (type) {
    case 'web': return <Server className={className} />;
    case 'static': return <Globe className={className} />;
    case 'postgres': return <Database className={className} />;
    case 'cron': return <Clock className={className} />;
    case 'blueprint': return <Layers className={className} />;
    default: return <Server className={className} />;
  }
};

// Reusable Dropdown Menu Component
const DropdownMenu = ({ trigger, items, right = false }) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative inline-block text-left">
      <div onClick={(e) => { e.stopPropagation(); setIsOpen(!isOpen); }}>
        {trigger}
      </div>

      {isOpen && (
        <>
          <div 
            className="fixed inset-0 z-40" 
            onClick={(e) => { e.stopPropagation(); setIsOpen(false); }} 
          />
          <div className={`absolute z-50 mt-2 w-48 rounded-lg bg-white border border-gray-200 shadow-lg ${right ? 'right-0 origin-top-right' : 'left-0 origin-top-left'}`}>
            <div className="py-1">
              {items.map((item, idx) => (
                item.divider ? (
                  <div key={`div-${idx}`} className="h-px bg-gray-200 my-1" />
                ) : (
                  <button
                    key={idx}
                    onClick={(e) => {
                      e.stopPropagation();
                      setIsOpen(false);
                      if(item.onClick) item.onClick();
                    }}
                    className={`w-full text-left px-4 py-2 text-sm flex items-center gap-2 transition-colors ${
                      item.danger ? 'text-red-600 hover:bg-red-50' : 'text-gray-700 hover:bg-gray-50'
                    }`}
                  >
                    {item.icon && <item.icon className="w-4 h-4" />}
                    {item.label}
                  </button>
                )
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
};

const SystemStats = () => {
  const stats = [
    { label: 'CPU Usage', value: '45%', icon: Cpu, color: 'bg-blue-500' },
    { label: 'Memory', value: '12.4 / 16 GB', icon: Activity, color: 'bg-indigo-500', width: '75%' },
    { label: 'Storage', value: '256 / 512 GB', icon: HardDrive, color: 'bg-green-500', width: '50%' },
    { label: 'Network I/O', value: '24.5 MB/s', icon: Wifi, color: 'bg-sky-500', width: '30%' },
  ];

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      {stats.map((stat, i) => (
        <div key={i} className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
          <div className="flex justify-between items-start mb-4">
            <div className="text-sm font-medium text-gray-500">{stat.label}</div>
            <div className="p-2 bg-gray-50 rounded-lg text-gray-400">
              <stat.icon className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold text-gray-900 mb-3">{stat.value}</div>
          <div className="h-2 w-full bg-gray-100 rounded-full overflow-hidden">
            <div 
              className={`h-full ${stat.color} rounded-full`} 
              style={{ width: stat.width || stat.value }} 
            />
          </div>
        </div>
      ))}
    </div>
  );
};

const Dashboard = ({ onSelectService, onNewService }) => {
  const [search, setSearch] = useState('');

  // Group services
  const groupedServices = MOCK_SERVICES.reduce((acc, service) => {
    if (!acc[service.group]) acc[service.group] = [];
    acc[service.group].push(service);
    return acc;
  }, {});

  const serviceActions = [
    { label: 'Restart', icon: RefreshCw },
    { label: 'Suspend', icon: Square },
    { divider: true },
    { label: 'Delete', icon: Trash2, danger: true }
  ];

  return (
    <div className="p-6 md:p-10 max-w-7xl mx-auto w-full">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-8 gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900 tracking-tight">Overview</h1>
          <p className="text-gray-500 mt-1">System health and active deployments.</p>
        </div>
        <button 
          onClick={onNewService}
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors flex items-center justify-center gap-2 shadow-sm"
        >
          <Plus className="w-4 h-4" />
          New Service
        </button>
      </div>

      <SystemStats />

      <div className="flex items-center gap-3 mb-6 bg-white p-2 border border-gray-200 rounded-xl shadow-sm">
        <Search className="w-5 h-5 ml-2 text-gray-400" />
        <input 
          type="text" 
          placeholder="Search services across all groups..." 
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full bg-transparent border-none focus:ring-0 text-sm text-gray-900 placeholder-gray-500 py-2 outline-none"
        />
      </div>

      <div className="space-y-8">
        {Object.entries(groupedServices).map(([groupName, services]) => {
          const filtered = services.filter(s => s.name.toLowerCase().includes(search.toLowerCase()));
          if (filtered.length === 0) return null;

          return (
            <div key={groupName} className="space-y-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-gray-700 px-1">
                <Folder className="w-4 h-4 text-blue-500" />
                {groupName}
                <span className="bg-gray-100 text-gray-500 px-2 py-0.5 rounded-full text-xs ml-2">
                  {filtered.length}
                </span>
              </div>
              
              <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
                <div className="divide-y divide-gray-100">
                  {filtered.map((service) => (
                    <div 
                      key={service.id}
                      onClick={() => onSelectService(service)}
                      className="p-4 sm:p-5 hover:bg-gray-50 transition-colors cursor-pointer group flex flex-col sm:flex-row sm:items-center justify-between gap-4"
                    >
                      <div className="flex items-start gap-4 flex-1">
                        <div className="p-2 bg-gray-50 rounded-lg text-blue-600 mt-1 sm:mt-0 border border-gray-200 group-hover:bg-blue-50 transition-colors">
                          <TypeIcon type={service.type} />
                        </div>
                        <div>
                          <div className="flex items-center gap-3 mb-1">
                            <h3 className="font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">
                              {service.name}
                            </h3>
                            <StatusBadge status={service.status} />
                          </div>
                          <div className="flex flex-wrap items-center text-sm text-gray-500 gap-x-4 gap-y-2">
                            <span className="flex items-center gap-1.5">
                              <Github className="w-3.5 h-3.5" />
                              {service.repo}
                            </span>
                            <span className="flex items-center gap-1.5">
                              <Clock className="w-3.5 h-3.5" />
                              Updated {service.lastDeploy}
                            </span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center justify-between sm:justify-end gap-6 w-full sm:w-auto pl-12 sm:pl-0">
                        <div className="text-sm text-gray-500 hidden md:block text-right">
                          <div className="font-medium text-gray-700">{service.region}</div>
                          <div>{service.runtime}</div>
                        </div>
                        <div className="flex items-center gap-2">
                          <DropdownMenu 
                            right 
                            trigger={
                              <button className="p-2 text-gray-400 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors">
                                <MoreVertical className="w-4 h-4" />
                              </button>
                            }
                            items={serviceActions}
                          />
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

const ContainersView = () => {
  return (
    <div className="p-6 md:p-10 max-w-7xl mx-auto w-full">
      <div className="mb-8">
        <h1 className="text-2xl font-semibold text-gray-900 tracking-tight">Docker Containers</h1>
        <p className="text-gray-500 mt-1">Manage underlying container instances.</p>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-gray-50 border-b border-gray-200 text-gray-600">
              <tr>
                <th className="px-6 py-4 font-medium">Container Name</th>
                <th className="px-6 py-4 font-medium">Image</th>
                <th className="px-6 py-4 font-medium">Status</th>
                <th className="px-6 py-4 font-medium">Ports</th>
                <th className="px-6 py-4 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {MOCK_CONTAINERS.map(container => (
                <tr key={container.id} className="hover:bg-gray-50 transition-colors">
                  <td className="px-6 py-4 font-medium text-gray-900 flex items-center gap-3">
                    <Box className="w-4 h-4 text-blue-500" />
                    {container.name}
                  </td>
                  <td className="px-6 py-4 font-mono text-gray-500">{container.image}</td>
                  <td className="px-6 py-4">
                    <StatusBadge status={container.status} />
                  </td>
                  <td className="px-6 py-4 text-gray-500">{container.ports}</td>
                  <td className="px-6 py-4 text-right flex justify-end gap-2">
                    <button className="p-1.5 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded transition-colors" title="Start">
                      <Play className="w-4 h-4" />
                    </button>
                    <button className="p-1.5 text-gray-400 hover:text-gray-900 hover:bg-gray-100 rounded transition-colors" title="Restart">
                      <RefreshCw className="w-4 h-4" />
                    </button>
                    <button className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors" title="Stop">
                      <Square className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

const BlueprintsView = () => {
  return (
    <div className="p-6 md:p-10 max-w-7xl mx-auto w-full">
       <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-8 gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900 tracking-tight">Blueprints</h1>
          <p className="text-gray-500 mt-1">Reusable infrastructure templates.</p>
        </div>
        <button className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors flex items-center gap-2 shadow-sm">
          <Plus className="w-4 h-4" />
          Create Blueprint
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {MOCK_BLUEPRINTS.map(bp => (
          <div key={bp.id} className="bg-white border border-gray-200 rounded-xl p-6 shadow-sm flex flex-col hover:border-blue-300 hover:shadow-md transition-all group">
            <div className="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center mb-4 border border-blue-100 group-hover:bg-blue-600 transition-colors">
              <Layers className="w-5 h-5 text-blue-600 group-hover:text-white transition-colors" />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 mb-1">{bp.name}</h3>
            <p className="text-sm text-gray-500 mb-4 flex-1">{bp.description}</p>
            <div className="pt-4 border-t border-gray-100 flex items-center justify-between text-sm">
               <span className="text-gray-400 flex items-center gap-1.5">
                  <FileCode className="w-4 h-4" /> {bp.source}
               </span>
               <button className="text-blue-600 font-medium hover:text-blue-700">Deploy</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

const ServiceDetailContent = ({ service, activeTab }) => {
  const [showEnv, setShowEnv] = useState(false);
  const [logSearch, setLogSearch] = useState('');
  const [isFollowingLogs, setIsFollowingLogs] = useState(true);
  
  const headerActions = [
    { label: 'Manual Deploy', icon: Play },
    { label: 'Restart Service', icon: RefreshCw },
    { divider: true },
    { label: 'Delete Service', icon: Trash2, danger: true }
  ];

  const envActions = [
    { label: 'Edit', icon: Edit2 },
    { label: 'Delete', icon: Trash2, danger: true }
  ];

  return (
    <div className="flex flex-col h-full bg-gray-50">
      {/* Detail Header */}
      <header className="border-b border-gray-200 bg-white pt-6 pb-6 px-6 md:px-10 z-10 shadow-sm">
        <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
          <div className="flex items-start gap-4">
            <div className="p-3 bg-blue-50 border border-blue-100 rounded-xl text-blue-600 shadow-sm">
              <TypeIcon type={service.type} className="w-8 h-8" />
            </div>
            <div>
              <div className="flex items-center gap-2 mb-1">
                <Folder className="w-4 h-4 text-gray-400" />
                <span className="text-sm font-medium text-gray-500">{service.group}</span>
              </div>
              <h1 className="text-2xl font-bold text-gray-900 mb-2 flex items-center gap-3 tracking-tight">
                {service.name}
                <StatusBadge status={service.status} />
              </h1>
              {service.url !== '-' && (
                <a href={`https://${service.url}`} target="_blank" rel="noreferrer" className="text-blue-600 hover:text-blue-700 text-sm flex items-center gap-1.5 transition-colors font-medium">
                  {service.url}
                  <ExternalLink className="w-3.5 h-3.5" />
                </a>
              )}
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <button className="bg-white hover:bg-gray-50 text-gray-700 px-4 py-2 rounded-lg text-sm font-medium transition-colors border border-gray-300 shadow-sm">
              Manual Deploy
            </button>
            <DropdownMenu 
              right
              trigger={
                <button className="bg-white hover:bg-gray-50 text-gray-500 p-2 rounded-lg transition-colors border border-gray-300 shadow-sm">
                  <MoreVertical className="w-5 h-5" />
                </button>
              }
              items={headerActions}
            />
          </div>
        </div>
      </header>

      {/* Tab Content Area */}
      <div className="flex-1 overflow-y-auto p-6 md:p-10">
        <div className="max-w-5xl mx-auto w-full">
          
          {}
          {activeTab === 'events' && (
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Deployment History</h3>
              {[1, 2, 3].map((_, i) => (
                <div key={i} className="bg-white border border-gray-200 rounded-xl p-5 flex items-start gap-4 shadow-sm hover:shadow-md transition-shadow">
                  <div className="mt-1">
                    {i === 0 && service.status === 'deploying' ? (
                      <div className="w-3 h-3 bg-yellow-400 rounded-full animate-pulse shadow-[0_0_8px_rgba(250,204,21,0.5)] ring-4 ring-yellow-50" />
                    ) : (
                      <div className="w-3 h-3 bg-green-500 rounded-full ring-4 ring-green-50" />
                    )}
                  </div>
                  <div className="flex-1">
                    <div className="flex justify-between items-start mb-1">
                      <h4 className="text-gray-900 font-medium">
                        {i === 0 && service.status === 'deploying' ? 'Deploy started' : 'Deploy succeeded'}
                      </h4>
                      <span className="text-xs text-gray-500 font-medium">
                        {i === 0 ? 'Just now' : `${i * 2} days ago`}
                      </span>
                    </div>
                    <p className="text-sm text-gray-600 mb-3">Triggered by commit <span className="font-mono text-gray-800 bg-gray-100 px-1.5 py-0.5 rounded border border-gray-200">a1b2c3d</span> on <span className="font-mono text-gray-800 bg-gray-100 px-1.5 py-0.5 rounded border border-gray-200">main</span></p>
                    <div className="text-xs font-mono text-gray-600 bg-gray-50 p-3 rounded-lg border border-gray-200 inline-block">
                      Update dependency packages and fix styling
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {}
          {activeTab === 'logs' && (
            <div className="h-[600px] bg-[#0c0c0c] rounded-xl border border-gray-800 flex flex-col overflow-hidden font-mono text-sm shadow-xl">
              <div className="bg-[#1a1a1a] border-b border-gray-800 px-4 py-2.5 flex flex-wrap gap-4 items-center justify-between text-gray-400">
                <div className="flex items-center gap-4">
                  <div className="flex items-center gap-2 bg-[#2a2a2a] px-3 py-1.5 rounded-lg border border-gray-700 text-gray-300 focus-within:border-blue-500 focus-within:ring-1 focus-within:ring-blue-500 transition-all">
                    <Search className="w-4 h-4" />
                    <input 
                      type="text" 
                      placeholder="Search logs..." 
                      className="bg-transparent border-none outline-none text-sm w-48 placeholder-gray-500" 
                      value={logSearch} 
                      onChange={e => setLogSearch(e.target.value)} 
                    />
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <button 
                    onClick={() => setIsFollowingLogs(!isFollowingLogs)} 
                    className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${isFollowingLogs ? 'text-blue-400 bg-blue-400/10 hover:bg-blue-400/20' : 'hover:bg-gray-800 hover:text-white'}`}
                  >
                    {isFollowingLogs ? <StopCircle className="w-4 h-4"/> : <PlayCircle className="w-4 h-4"/>}
                    {isFollowingLogs ? 'Following' : 'Paused'}
                  </button>
                  <div className="w-px h-5 bg-gray-700"></div>
                  <button className="flex items-center gap-1.5 text-sm font-medium px-3 py-1.5 hover:bg-gray-800 hover:text-white rounded-lg transition-colors">
                    <Trash2 className="w-4 h-4"/> Clear
                  </button>
                </div>
              </div>
              <div className="p-4 overflow-y-auto flex-1 text-gray-300 space-y-1.5 text-xs leading-relaxed font-mono">
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:00Z</span><span className="text-blue-400">==&gt; Starting build process...</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:02Z</span><span className="text-gray-300">Cloning repository {service.repo}...</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:05Z</span><span className="text-green-400">✓ Repository cloned successfully</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:06Z</span><span className="text-gray-300">Checking out branch {service.branch}...</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:07Z</span><span className="text-gray-300">Running build command 'npm run build'...</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:09Z</span><span className="text-purple-400">&gt; my-app@1.0.0 build</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:12Z</span><span className="text-purple-400">&gt; react-scripts build</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:18Z</span><span className="text-gray-300">Creating an optimized production build...</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:45Z</span><span className="text-gray-300">Compiled successfully.</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:46Z</span><span className="text-green-400">==&gt; Build successful 🎉</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:48Z</span><span className="text-blue-400">==&gt; Starting service with 'npm start'...</span></div>
                <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:50Z</span><span className="text-gray-300">Server running on port 10000</span></div>
                {service.status === 'deploying' && (
                   <div className="flex gap-4 hover:bg-white/5 px-2 py-0.5 rounded transition-colors -mx-2"><span className="text-gray-600 shrink-0 select-none">2026-07-29T01:50:55Z</span><span className="text-yellow-400 animate-pulse">Waiting for health check...</span></div>
                )}
              </div>
            </div>
          )}

          {}
          {activeTab === 'env' && (
            <div className="space-y-6">
              <div className="flex justify-between items-center">
                <div>
                  <h3 className="text-lg font-medium text-gray-900">Environment Variables</h3>
                  <p className="text-sm text-gray-500 mt-1">Manage configuration for your service.</p>
                </div>
                <div className="flex items-center gap-3">
                  <button 
                    onClick={() => setShowEnv(!showEnv)}
                    className="bg-white hover:bg-gray-50 text-gray-700 border border-gray-300 px-3 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm flex items-center gap-2"
                  >
                    {showEnv ? <EyeOff className="w-4 h-4"/> : <Eye className="w-4 h-4"/>}
                    {showEnv ? 'Hide Values' : 'Reveal Values'}
                  </button>
                  <button className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm flex items-center gap-2">
                    <Plus className="w-4 h-4"/> Add Variable
                  </button>
                </div>
              </div>

              <div className="bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm">
                <div className="grid grid-cols-12 gap-4 p-4 border-b border-gray-200 text-sm font-medium text-gray-600 bg-gray-50">
                  <div className="col-span-4">Key</div>
                  <div className="col-span-6">Value</div>
                  <div className="col-span-2"></div>
                </div>
                <div className="divide-y divide-gray-100">
                  {['DATABASE_URL', 'NODE_ENV', 'API_KEY'].map((key) => (
                    <div key={key} className="grid grid-cols-12 gap-4 p-4 items-center group hover:bg-gray-50 transition-colors">
                      <div className="col-span-4 font-mono text-sm text-gray-900">{key}</div>
                      <div className="col-span-6 font-mono text-sm text-gray-500 truncate">
                        {(showEnv || key === 'NODE_ENV') ? (key === 'NODE_ENV' ? 'production' : 'super_secret_value_123') : '************************'}
                      </div>
                      <div className="col-span-2 flex justify-end gap-2">
                        <button className="text-gray-400 hover:text-blue-600 p-1.5 rounded hover:bg-blue-50 transition-colors opacity-0 group-hover:opacity-100" title="Copy value">
                           <Copy className="w-4 h-4" />
                        </button>
                        <DropdownMenu 
                          right
                          trigger={
                            <button className="text-gray-400 hover:text-gray-700 p-1.5 rounded hover:bg-gray-200 transition-colors opacity-0 group-hover:opacity-100">
                              <MoreVertical className="w-4 h-4"/>
                            </button>
                          }
                          items={envActions}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {}
          {activeTab === 'domains' && (
            <div className="space-y-6 max-w-4xl">
              <div className="flex justify-between items-center">
                <div>
                  <h3 className="text-lg font-medium text-gray-900">Custom Domains</h3>
                  <p className="text-sm text-gray-500 mt-1">Manage custom domains and SSL certificates for your service.</p>
                </div>
                <button className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm flex items-center gap-2">
                  <Plus className="w-4 h-4"/> Add Domain
                </button>
              </div>

              {/* Default Domain */}
              <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
                <div className="flex items-center justify-between mb-2">
                   <div className="flex items-center gap-3">
                     <div className="p-2 bg-gray-50 border border-gray-200 rounded-lg"><Globe className="w-5 h-5 text-gray-500" /></div>
                     <div>
                       <h4 className="font-medium text-gray-900">{service.url}</h4>
                       <span className="inline-block mt-1 text-xs text-gray-500 bg-gray-100 px-2 py-0.5 rounded-full border border-gray-200">Default Render URL</span>
                     </div>
                   </div>
                   <a href={`https://${service.url}`} target="_blank" rel="noreferrer" className="text-blue-600 hover:text-blue-700 text-sm font-medium flex items-center gap-1 px-3 py-1.5 bg-blue-50 rounded-lg transition-colors">
                      Visit <ExternalLink className="w-3.5 h-3.5" />
                   </a>
                </div>
              </div>

              {/* Custom Domains list */}
              <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
                <div className="p-5 border-b border-gray-100 flex items-center justify-between">
                   <div>
                     <h4 className="font-medium text-gray-900 flex items-center gap-2">www.myapp.com <CheckCircle className="w-4 h-4 text-green-500" /></h4>
                     <p className="text-sm text-gray-500 mt-1">Managed TLS enabled. Last renewed 12 days ago.</p>
                   </div>
                   <DropdownMenu 
                      right
                      trigger={<button className="p-2 text-gray-400 hover:bg-gray-100 rounded-lg"><MoreVertical className="w-5 h-5" /></button>}
                      items={[
                        { label: 'Verify DNS', icon: RefreshCw },
                        { divider: true },
                        { label: 'Remove Domain', icon: Trash2, danger: true }
                      ]}
                   />
                </div>
                <div className="p-5 bg-gray-50/50 text-sm">
                   <p className="text-gray-700 font-medium mb-3">DNS Configuration</p>
                   <div className="grid grid-cols-12 gap-4 text-gray-500 font-mono text-xs border border-gray-200 rounded-lg p-3 bg-white shadow-sm">
                      <div className="col-span-2 font-semibold text-gray-700 uppercase tracking-wider text-[10px]">Type</div>
                      <div className="col-span-4 font-semibold text-gray-700 uppercase tracking-wider text-[10px]">Name</div>
                      <div className="col-span-6 font-semibold text-gray-700 uppercase tracking-wider text-[10px]">Value</div>
                      
                      <div className="col-span-12 h-px bg-gray-100 -mx-3"></div>

                      <div className="col-span-2 flex items-center">CNAME</div>
                      <div className="col-span-4 flex items-center">www</div>
                      <div className="col-span-6 flex items-center justify-between break-all">
                        {service.url}
                        <button className="text-gray-400 hover:text-blue-600 transition-colors"><Copy className="w-3.5 h-3.5"/></button>
                      </div>
                   </div>
                </div>
              </div>
            </div>
          )}

          {}
          {activeTab === 'metrics' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {['CPU Usage', 'Memory Usage', 'Bandwidth', 'Requests'].map(metric => (
                <div key={metric} className="bg-white border border-gray-200 rounded-xl p-5 h-64 flex flex-col shadow-sm">
                  <h4 className="text-sm font-medium text-gray-700 mb-4">{metric}</h4>
                  <div className="flex-1 flex items-center justify-center text-gray-400 border-2 border-dashed border-gray-100 rounded-lg bg-gray-50">
                    <Activity className="w-6 h-6 mr-2 opacity-40 text-blue-500"/>
                    <span className="text-sm font-medium">No data available yet</span>
                  </div>
                </div>
              ))}
            </div>
          )}

          {}
          {activeTab === 'settings' && (
            <div className="space-y-8 max-w-3xl">
              <section>
                <h3 className="text-lg font-medium text-gray-900 mb-4">General Settings</h3>
                <div className="bg-white border border-gray-200 rounded-xl p-6 space-y-5 shadow-sm">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1.5">Service Name</label>
                    <input type="text" defaultValue={service.name} className="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-colors shadow-sm" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1.5">Build Command</label>
                    <input type="text" defaultValue="npm run build" className="w-full font-mono bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-colors shadow-sm" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1.5">Start Command</label>
                    <input type="text" defaultValue="npm start" className="w-full font-mono bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-colors shadow-sm" />
                  </div>
                  <div className="pt-2">
                    <button className="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2.5 rounded-lg text-sm font-medium transition-colors shadow-sm">
                      Save Changes
                    </button>
                  </div>
                </div>
              </section>
              
              <section>
                <h3 className="text-lg font-medium text-red-600 mb-4">Danger Zone</h3>
                <div className="bg-white border border-red-200 rounded-xl p-6 shadow-sm">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div>
                      <h4 className="text-gray-900 font-medium mb-1">Delete Service</h4>
                      <p className="text-sm text-gray-500">Permanently remove this service and all its data. This cannot be undone.</p>
                    </div>
                    <button className="bg-red-50 hover:bg-red-100 text-red-600 border border-red-200 px-5 py-2.5 rounded-lg text-sm font-medium transition-colors whitespace-nowrap">
                      Delete Service
                    </button>
                  </div>
                </div>
              </section>
            </div>
          )}

        </div>
      </div>
    </div>
  );
};

const NewService = ({ onBack, onDeploy }) => {
  const [step, setStep] = useState(1);
  const [selectedType, setSelectedType] = useState(null);
  const [selectedRepo, setSelectedRepo] = useState(null);

  const SERVICE_TYPES = [
    { id: 'blueprint', title: 'Deploy Blueprint', desc: 'Use pre-configured infra templates', icon: Layers, color: 'text-blue-600', bg: 'bg-blue-50', border: 'border-blue-200', needsRepo: false },
    { id: 'web', title: 'Web Service', desc: 'Node, Python, Go, Ruby, Docker', icon: Server, color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true },
    { id: 'static', title: 'Static Site', desc: 'React, Vue, Astro, HTML/CSS', icon: Globe, color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true },
    { id: 'postgres', title: 'PostgreSQL', desc: 'Managed relational database', icon: Database, color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: false },
    { id: 'redis', title: 'Redis', desc: 'Managed in-memory cache', icon: Database, color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: false },
    { id: 'cron', title: 'Cron Job', desc: 'Scheduled tasks and scripts', icon: Clock, color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true },
  ];

  const handleTypeSelect = (type) => {
    setSelectedType(type);
    if (type.needsRepo) {
      setStep(2);
    } else {
      setStep(3);
    }
  };

  const handleRepoSelect = (repo) => {
    setSelectedRepo(repo);
    setStep(3);
  };

  const renderSpecificFields = () => {
    switch (selectedType?.id) {
      case 'web': return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">Runtime</label>
            <select className="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500">
              <option>Node.js</option>
              <option>Python</option>
              <option>Docker</option>
              <option>Go</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">Build Command</label>
            <input type="text" defaultValue="npm install && npm run build" className="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">Start Command</label>
            <input type="text" defaultValue="npm start" className="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
          </div>
        </>
      );
      case 'static': return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">Build Command</label>
            <input type="text" defaultValue="npm run build" className="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">Publish Directory</label>
            <input type="text" defaultValue="dist" className="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
          </div>
        </>
      );
      case 'postgres': return (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">PostgreSQL Version</label>
          <select className="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500">
            <option>15</option>
            <option>14</option>
            <option>13</option>
          </select>
        </div>
      );
      case 'redis': return (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">Redis Version</label>
          <select className="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500">
            <option>7.0</option>
            <option>6.2</option>
          </select>
        </div>
      );
      case 'cron': return (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">Schedule (Cron Expression)</label>
            <input type="text" defaultValue="0 0 * * *" className="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">Command</label>
            <input type="text" defaultValue="python sync.py" className="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
          </div>
        </>
      );
      case 'blueprint': return (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">Select Blueprint</label>
          <select className="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500">
            {MOCK_BLUEPRINTS.map(bp => (
              <option key={bp.id}>{bp.name}</option>
            ))}
          </select>
        </div>
      );
      default: return null;
    }
  };

  return (
    <div className="p-6 md:p-10 max-w-5xl mx-auto w-full">
      <button 
        onClick={() => {
          if (step > 1) {
            if (step === 3 && selectedType?.needsRepo) setStep(2);
            else setStep(1);
          } else {
            onBack();
          }
        }}
        className="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-900 transition-colors mb-8 font-medium"
      >
        <ArrowLeft className="w-4 h-4" />
        {step === 1 ? 'Back to Dashboard' : 'Back'}
      </button>

      {/* Step 1: Select Type */}
      {step === 1 && (
        <div className="animate-in fade-in slide-in-from-bottom-4 duration-500">
          <h1 className="text-3xl font-bold text-gray-900 mb-2 tracking-tight">Deploy a New Resource</h1>
          <p className="text-gray-500 mb-10 text-lg">Select a service type or blueprint to deploy to your infrastructure.</p>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {SERVICE_TYPES.map((item, i) => (
              <div 
                key={i} 
                onClick={() => handleTypeSelect(item)}
                className={`bg-white border ${item.border} hover:border-blue-300 hover:shadow-md rounded-xl p-6 cursor-pointer transition-all hover:-translate-y-1 group flex flex-col h-full`}
              >
                <div className={`w-12 h-12 rounded-lg ${item.bg} ${item.color} flex items-center justify-center mb-6 transition-colors group-hover:bg-blue-600 group-hover:text-white`}>
                  <item.icon className="w-6 h-6" />
                </div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2 group-hover:text-blue-600 transition-colors">{item.title}</h3>
                <p className="text-gray-500 text-sm flex-1">{item.desc}</p>
                <div className="mt-6 flex items-center gap-2 text-sm font-medium text-gray-400 group-hover:text-blue-600 transition-colors">
                  Continue <ChevronRight className="w-4 h-4 opacity-0 group-hover:opacity-100 -ml-2 group-hover:ml-0 transition-all" />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Step 2: Select Source */}
      {step === 2 && (
        <div className="max-w-3xl mx-auto animate-in fade-in slide-in-from-bottom-4 duration-500">
          <h2 className="text-2xl font-bold text-gray-900 mb-6">Connect a repository</h2>
          <div className="bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm">
            <div className="p-4 border-b border-gray-200 bg-gray-50 flex items-center justify-between">
               <div className="flex items-center gap-2 text-sm font-medium text-gray-700">
                 <Github className="w-5 h-5" /> GitHub
               </div>
               <span className="text-sm text-gray-500">my-org</span>
            </div>
            <div className="divide-y divide-gray-100">
              {MOCK_REPOS.map(repo => (
                <div key={repo.id} className="p-4 flex items-center justify-between hover:bg-gray-50 transition-colors">
                  <div className="flex items-center gap-3">
                    {repo.private ? <Lock className="w-4 h-4 text-gray-400" /> : <Unlock className="w-4 h-4 text-gray-400" />}
                    <div>
                      <div className="font-medium text-gray-900">{repo.name}</div>
                      <div className="text-xs text-gray-500">Updated {repo.updated}</div>
                    </div>
                  </div>
                  <button 
                    onClick={() => handleRepoSelect(repo)}
                    className="px-4 py-1.5 bg-white border border-gray-300 text-gray-700 text-sm font-medium rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
                  >
                    Connect
                  </button>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Step 3: Configure */}
      {step === 3 && selectedType && (
        <div className="max-w-3xl mx-auto animate-in fade-in slide-in-from-bottom-4 duration-500">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Configure {selectedType.title}</h2>
          {selectedRepo && (
             <p className="text-gray-500 mb-6 flex items-center gap-2 text-sm">
               Deploying from <span className="font-mono bg-gray-100 text-gray-700 px-1.5 py-0.5 rounded border border-gray-200">{selectedRepo.name}</span>
             </p>
          )}
          {!selectedRepo && <div className="mb-6"></div>}
          
          <div className="bg-white border border-gray-200 rounded-xl p-6 md:p-8 shadow-sm space-y-6">
            {/* Common Fields */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Name</label>
                <input type="text" placeholder="e.g. my-awesome-app" className="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 shadow-sm" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Region</label>
                <select className="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 shadow-sm">
                  <option>Frankfurt (EU)</option>
                  <option>Ohio (US East)</option>
                  <option>Oregon (US West)</option>
                  <option>Singapore (Asia)</option>
                </select>
              </div>
            </div>

            {/* Branch (if repo) */}
            {selectedRepo && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Branch</label>
                <div className="relative">
                  <GitBranch className="w-5 h-5 text-gray-400 absolute left-3 top-2.5" />
                  <select className="w-full bg-white border border-gray-300 rounded-lg pl-10 pr-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 shadow-sm">
                    <option>main</option>
                    <option>staging</option>
                    <option>development</option>
                  </select>
                </div>
              </div>
            )}

            {/* Render type-specific fields */}
            {renderSpecificFields()}
            
            {/* Plan Selection */}
            <div className="pt-6 mt-6 border-t border-gray-100">
               <label className="block text-base font-medium text-gray-900 mb-4">Instance Type</label>
               <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                 {['Free', 'Starter ($7/mo)', 'Pro ($25/mo)'].map((plan, idx) => (
                   <label key={plan} className="relative border border-gray-200 rounded-xl p-4 flex flex-col cursor-pointer hover:border-blue-400 has-[:checked]:border-blue-600 has-[:checked]:bg-blue-50 transition-colors">
                     <input type="radio" name="plan" className="peer sr-only" defaultChecked={idx === 1} />
                     <span className="font-semibold text-gray-900">{plan.split(' ')[0]}</span>
                     <span className="text-sm text-gray-500 mt-1">{plan.includes('$') ? plan.split(' ')[1] : '$0/month'}</span>
                     <CheckCircle className="w-5 h-5 text-blue-600 absolute top-4 right-4 opacity-0 peer-checked:opacity-100 transition-opacity" />
                   </label>
                 ))}
               </div>
            </div>
          </div>

          <div className="mt-8 flex justify-end gap-4">
             <button 
               onClick={() => setStep(selectedType.needsRepo ? 2 : 1)} 
               className="px-5 py-2.5 text-gray-600 font-medium hover:bg-gray-100 rounded-lg transition-colors"
             >
               Back
             </button>
             <button 
               onClick={onDeploy} 
               className="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors shadow-sm"
             >
               Create {selectedType.title}
             </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default function App() {
  const [appView, setAppView] = useState('dashboard'); // 'dashboard', 'containers', 'blueprints', 'new'
  const [selectedService, setSelectedService] = useState(null);
  const [serviceTab, setServiceTab] = useState('events'); // 'events', 'logs', 'env', 'metrics', 'settings'
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleSelectService = (service) => {
    setSelectedService(service);
    setServiceTab('events');
    setAppView('detail'); // implicit state
    setMobileMenuOpen(false);
  };

  const handleBackToSystem = () => {
    setSelectedService(null);
    setAppView('dashboard');
    setMobileMenuOpen(false);
  };

  const navigateTo = (view) => {
    setAppView(view);
    setSelectedService(null);
    setMobileMenuOpen(false);
  };

  const globalNav = [
    { id: 'dashboard', icon: LayoutDashboard, label: 'Overview' },
    { id: 'containers', icon: Box, label: 'Containers' },
    { id: 'blueprints', icon: Layers, label: 'Blueprints' },
    { id: 'workspaces', icon: Briefcase, label: 'Workspaces' },
  ];

  const serviceNav = [
    { id: 'events', icon: Activity, label: 'Events' },
    { id: 'logs', icon: Terminal, label: 'Logs' },
    { id: 'env', icon: Database, label: 'Environment' },
    { id: 'domains', icon: Globe, label: 'Custom Domains' },
    { id: 'metrics', icon: Activity, label: 'Metrics' },
    { id: 'settings', icon: Settings, label: 'Settings' },
  ];

  const renderSidebarContent = () => (
    <>
      <div className="h-16 flex items-center px-6 border-b border-gray-200">
        <div className="flex items-center gap-2 text-gray-900 font-bold text-lg tracking-tight cursor-pointer" onClick={handleBackToSystem}>
          <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center shadow-sm">
            <Cpu className="w-5 h-5 text-white" />
          </div>
          DevPanel
        </div>
      </div>

      <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
        {selectedService ? (
          // Service Specific Sidebar
          <>
            <button
              onClick={handleBackToSystem}
              className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium text-gray-500 hover:text-gray-900 hover:bg-gray-100 transition-colors mb-4"
            >
              <ArrowLeft className="w-4 h-4" />
              Back to System
            </button>
            <div className="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2 mt-4">
              {selectedService.name}
            </div>
            {serviceNav.map((item) => (
              <button
                key={item.id}
                onClick={() => setServiceTab(item.id)}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  serviceTab === item.id
                    ? 'bg-blue-50 text-blue-700'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                }`}
              >
                <item.icon className={`w-4 h-4 ${serviceTab === item.id ? 'text-blue-600' : 'text-gray-400'}`} />
                {item.label}
              </button>
            ))}
          </>
        ) : (
          // Global System Sidebar
          <>
            <div className="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
              System
            </div>
            {globalNav.map((item) => (
              <button
                key={item.id}
                onClick={() => navigateTo(item.id)}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  appView === item.id
                    ? 'bg-blue-50 text-blue-700'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                }`}
              >
                <item.icon className={`w-4 h-4 ${appView === item.id ? 'text-blue-600' : 'text-gray-400'}`} />
                {item.label}
              </button>
            ))}
          </>
        )}
      </nav>

      <div className="p-4 border-t border-gray-200 bg-gray-50/50">
        <button className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium text-gray-600 hover:bg-gray-100 hover:text-gray-900 transition-colors">
          <Settings className="w-4 h-4 text-gray-400" />
          User Settings
        </button>
        <div className="mt-4 px-3 flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-blue-100 border border-blue-200 flex items-center justify-center text-blue-700 font-bold text-sm shadow-sm">
            JD
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-gray-900 truncate">John Doe</p>
            <p className="text-xs text-gray-500 truncate">Personal Account</p>
          </div>
        </div>
      </div>
    </>
  );

  return (
    <div className="min-h-screen bg-gray-50 text-gray-900 font-sans flex h-screen overflow-hidden selection:bg-blue-200 selection:text-blue-900">
      
      {/* Desktop Sidebar */}
      <aside className="w-64 border-r border-gray-200 bg-white hidden md:flex flex-col shrink-0 z-20">
        {renderSidebarContent()}
      </aside>

      {/* Mobile Sidebar Overlay */}
      {mobileMenuOpen && (
        <div className="md:hidden fixed inset-0 z-50 flex">
          <div className="fixed inset-0 bg-gray-900/50" onClick={() => setMobileMenuOpen(false)} />
          <aside className="w-72 max-w-[80%] bg-white h-full flex flex-col relative z-50 shadow-2xl">
            <button 
              className="absolute top-4 right-4 p-2 text-gray-500 hover:bg-gray-100 rounded-lg"
              onClick={() => setMobileMenuOpen(false)}
            >
              <X className="w-5 h-5" />
            </button>
            {renderSidebarContent()}
          </aside>
        </div>
      )}

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col min-w-0 bg-gray-50 overflow-hidden relative">
        
        {/* Mobile Header */}
        <div className="md:hidden h-16 border-b border-gray-200 bg-white flex items-center justify-between px-4 shrink-0 z-10">
          <div className="flex items-center gap-3">
            <button className="text-gray-600 p-1 hover:bg-gray-100 rounded-lg" onClick={() => setMobileMenuOpen(true)}>
              <Menu className="w-6 h-6" />
            </button>
            <div className="flex items-center gap-2 text-gray-900 font-bold text-lg tracking-tight">
              <div className="w-7 h-7 bg-blue-600 rounded-md flex items-center justify-center shadow-sm">
                <Cpu className="w-4 h-4 text-white" />
              </div>
              DevPanel
            </div>
          </div>
          <DropdownMenu 
            right
            trigger={
              <button className="text-gray-500 hover:text-gray-900 p-2 rounded-lg hover:bg-gray-100">
                <MoreVertical className="w-5 h-5" />
              </button>
            }
            items={[
              { label: 'New Service', icon: Plus },
              { label: 'Settings', icon: Settings },
            ]}
          />
        </div>

        {/* Dynamic View Rendering */}
        <div className="flex-1 overflow-y-auto">
          {(!selectedService && appView === 'dashboard') && (
            <Dashboard 
              onSelectService={handleSelectService} 
              onNewService={() => setAppView('new')}
            />
          )}
          
          {(!selectedService && appView === 'containers') && <ContainersView />}
          
          {(!selectedService && appView === 'blueprints') && <BlueprintsView />}

          {(!selectedService && appView === 'workspaces') && (
            <div className="p-10 flex flex-col items-center justify-center h-full text-center">
              <Briefcase className="w-12 h-12 text-gray-300 mb-4" />
              <h2 className="text-xl font-semibold text-gray-900">Workspaces</h2>
              <p className="text-gray-500 mt-2">Manage your workspaces, team access, and billing here.</p>
            </div>
          )}

          {selectedService && (
            <ServiceDetailContent 
              service={selectedService} 
              activeTab={serviceTab}
            />
          )}

          {appView === 'new' && (
             <NewService onBack={handleBackToSystem} onDeploy={handleBackToSystem} />
          )}
        </div>
      </main>
      
    </div>
  );
}