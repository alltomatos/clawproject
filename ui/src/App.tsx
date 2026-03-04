import { motion, AnimatePresence } from 'framer-motion';
import { useEffect, useRef, useState } from 'react';
import { LayoutDashboard, MessageSquare, Settings, Activity, Plus, Send, ArrowLeft, Bot, LayoutGrid, Table2, Trash2 } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface NavItemProps { icon: React.ReactNode; label: string; active?: boolean; onClick?: () => void; }
type View = 'dashboard' | 'projects' | 'newProject' | 'chat';
type ChatItem = { sender: 'agent' | 'user'; message: string };
type Project = { 
  id: string; 
  name: string; 
  description?: string; 
  manager_status: string; 
  manager_session_key: string;
  leader_name?: string;
  leader_email?: string;
  location?: string;
  vibe?: string;
  project_type?: 'new' | 'existing';
  git_url?: string;
};
type ManagerInfo = { manager_status: string; manager_enabled: boolean; manager_session_key: string; api_calls: number; daily_calls?: number };
type ProjectMessage = { id: string; sender: 'agent' | 'user'; message: string };
type PlannerInfo = { stage: string; project_type: string; niche: string; deliverables: string[]; deliverables_done: string[]; last_checkpoint: string };

const Dashboard = () => {
  const [view, setView] = useState<View>('dashboard');
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<string>('');
  const [displayMode, setDisplayMode] = useState<'grid' | 'table'>('grid');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'offline' | 'paused'>('all');

  const [message, setMessage] = useState('');
  const [chatHistory, setChatHistory] = useState<ChatItem[]>([{ sender: 'agent', message: 'Selecione um projeto e converse com o gestor.' }]);

  const [manager, setManager] = useState<ManagerInfo | null>(null);
  const [planner, setPlanner] = useState<PlannerInfo | null>(null);

  const [sending, setSending] = useState(false);
  const [controlling, setControlling] = useState(false);
  const [creatingProject, setCreatingProject] = useState(false);

  const [newName, setNewName] = useState('');
  const [newDescription, setNewDescription] = useState('');
  const [leaderName, setLeaderName] = useState('');
  const [leaderEmail, setLeaderEmail] = useState('');
  const [location, setLocation] = useState('');
  const [vibe, setVibe] = useState('Profissional e Pragmático');
  const [projectType, setProjectType] = useState<'new' | 'existing'>('new');
  const [gitUrl, setGitUrl] = useState('');

  const reconnectAttemptedRef = useRef<Record<string, boolean>>({});
  const version = '0.2.4-stable';

  const selectedProject = projects.find((p) => p.id === selectedProjectId) || null;
  const filteredProjects = projects.filter((p) => {
    if (statusFilter === 'all') return true;
    return p.manager_status.toLowerCase() === statusFilter.toLowerCase();
  });
  const managerOnline = manager?.manager_status === 'active';
  const plannerStages = ['triage_type', 'triage_niche', 'objective', 'deliverables', 'visual_checklist', 'active'];
  const stageLabels: Record<string, string> = { triage_type: 'Tipo', triage_niche: 'Nicho', objective: 'Objetivo', deliverables: 'Entregáveis', visual_checklist: 'Checklist Visual', active: 'Execução' };
  const stageIndex = planner ? Math.max(0, plannerStages.indexOf(planner.stage)) : 0;

  const loadProjects = async () => {
    try {
      const r = await fetch('/api/projects');
      if (!r.ok) return;
      const data: Project[] = await r.json();
      setProjects(data);
      if (!selectedProjectId && data.length) setSelectedProjectId(data[0].id);
    } catch (err) {
      console.error('Erro ao carregar projetos:', err);
    }
  };

  const loadManager = async (projectId: string) => {
    const r = await fetch(`/api/projects/${projectId}/manager`);
    if (!r.ok) return;
    setManager(await r.json());
  };

  const loadPlanner = async (projectId: string) => {
    const r = await fetch(`/api/projects/${projectId}/planner`);
    if (!r.ok) return;
    setPlanner(await r.json());
  };

  const loadMessages = async (projectId: string) => {
    const r = await fetch(`/api/projects/${projectId}/messages`);
    if (!r.ok) return;
    const data: ProjectMessage[] = await r.json();
    setChatHistory(data.length ? data.map((m) => ({ sender: m.sender, message: m.message })) : [{ sender: 'agent', message: 'Sem histórico ainda.' }]);
  };

  useEffect(() => { loadProjects(); }, []);

  useEffect(() => {
    const boot = async () => {
      if (!selectedProjectId) return;
      await Promise.all([loadMessages(selectedProjectId), loadManager(selectedProjectId), loadPlanner(selectedProjectId)]);
      if ((manager?.manager_status === 'offline' || manager?.manager_status === 'paused') && !reconnectAttemptedRef.current[selectedProjectId]) {
        reconnectAttemptedRef.current[selectedProjectId] = true;
        await fetch(`/api/projects/${selectedProjectId}/manager/control`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action: 'resume' }),
        });
        await loadManager(selectedProjectId);
      }
    };
    boot();
  }, [selectedProjectId]); // eslint-disable-line

  const createProject = async () => {
    if (!newName.trim() || creatingProject) return;
    setCreatingProject(true);
    try {
      const payload = { 
        name: newName.trim(), 
        description: newDescription.trim() || 'Projeto iniciado via ClawProject',
        leader_name: leaderName.trim(),
        leader_email: leaderEmail.trim(),
        location: location.trim(),
        vibe: vibe,
        project_type: projectType,
        git_url: gitUrl.trim()
      };
      const r = await fetch('/api/projects', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
      if (!r.ok) throw new Error();
      const created: Project = await r.json();
      await new Promise(resolve => setTimeout(resolve, 3500));
      await loadProjects();
      setSelectedProjectId(created.id);
      setView('chat');
      setNewName('');
      setNewDescription('');
    } finally {
      setCreatingProject(false);
    }
  };

  const openProject = (id: string) => {
    setSelectedProjectId(id);
    setView('chat');
  };

  const handleSendMessage = async () => {
    if (!message.trim() || sending || !selectedProjectId) return;
    const userText = message;
    setMessage('');
    setChatHistory((prev) => [...prev, { sender: 'user', message: userText }]);
    setSending(true);
    try {
      const r = await fetch(`/api/projects/${selectedProjectId}/manager/message`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ message: userText }),
      });
      if (!r.ok) throw new Error();
      await Promise.all([loadMessages(selectedProjectId), loadManager(selectedProjectId), loadPlanner(selectedProjectId), loadProjects()]);
    } catch {
      setChatHistory((prev) => [...prev, { sender: 'agent', message: 'Gestor indisponível no momento.' }]);
    } finally {
      setSending(false);
    }
  };

  const handleControl = async (action: 'restart' | 'pause' | 'resume' | 'start-execution') => {
    if (!selectedProjectId || controlling) return;
    setControlling(true);
    try {
      const r = await fetch(`/api/projects/${selectedProjectId}/manager/control`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action }),
      });
      
      if (action === 'start-execution' && r.ok) {
        // Se foi o START, damos um pequeno delay visual e vamos para a atividade/pipeline
        await new Promise(resolve => setTimeout(resolve, 2000));
        setView('projects'); // Por enquanto volta para projetos ou podemos ir para 'Atividade'
      }

      await Promise.all([loadManager(selectedProjectId), loadPlanner(selectedProjectId), loadProjects(), loadMessages(selectedProjectId)]);
    } finally {
      setControlling(false);
    }
  };

  const deleteProject = async (id: string, name: string) => {
    if (!window.confirm(`⚠️ DESTRUIÇÃO TOTAL: Tem certeza que deseja deletar o projeto "${name}"? \n\nIsso irá remover permanentemente:\n- Todos os arquivos da pasta\n- O agente no OpenClaw\n- O histórico de mensagens`)) return;
    
    setControlling(true); // Reutilizamos o estado de loading/bloqueio
    try {
      const r = await fetch(`/api/projects/delete/${id}`, { method: 'DELETE' });
      if (r.ok) {
        // Aguardamos 5 segundos para o Gateway do OpenClaw reiniciar e estabilizar
        await new Promise(resolve => setTimeout(resolve, 5000));
        await loadProjects();
        if (selectedProjectId === id) setSelectedProjectId('');
      }
    } catch (err) {
      alert('Erro ao deletar projeto');
    } finally {
      setControlling(false);
    }
  };

  return (
    <div className="flex h-screen w-full bg-[#F5F5F7] overflow-hidden text-[#1D1D1F]">
      <aside className="w-20 md:w-64 glass border-r border-gray-200 flex flex-col p-4 z-20">
        <div className="flex items-center space-x-3 px-2 mb-8 mt-2">
          <div className="w-10 h-10 bg-indigo-600 rounded-xl flex items-center justify-center text-white font-black text-xl">C</div>
          <span className="font-extrabold text-xl tracking-tighter hidden md:block">ClawProject</span>
        </div>

        <nav className="space-y-2">
          <NavItem icon={<LayoutDashboard size={20} />} label="Dashboard" active={view === 'dashboard'} onClick={() => setView('dashboard')} />
          <NavItem icon={<MessageSquare size={20} />} label="Projetos" active={view === 'projects' || view === 'newProject' || view === 'chat'} onClick={() => setView('projects')} />
          <NavItem icon={<Activity size={20} />} label="Atividade" />
        </nav>

        <div className="pt-4 border-t border-gray-100 mt-auto">
          <NavItem icon={<Settings size={20} />} label="Configurações" />
        </div>
      </aside>

      <main className="flex-1 flex flex-col overflow-hidden">
        <header className="h-20 glass border-b border-gray-200 flex items-center justify-between px-8">
          <div className="flex items-center space-x-4">
            {(view === 'chat' || view === 'newProject') && <button onClick={() => setView('projects')} className="p-2 hover:bg-gray-100 rounded-full"><ArrowLeft size={20} /></button>}
            <h2 className="text-2xl font-bold tracking-tight text-gray-900">
              {view === 'dashboard' ? 'Visão Geral' : view === 'projects' ? 'Projetos' : view === 'newProject' ? 'Novo Projeto' : 'Modo Planejador'}
            </h2>
          </div>
          <div className="text-right">
            <div className="text-[10px] font-black text-slate-400 uppercase">Gestor</div>
            <div className={`text-xs font-bold ${managerOnline ? 'text-emerald-600' : 'text-amber-600'}`}>{managerOnline ? 'ONLINE' : 'OFFLINE'}</div>
            <div className="text-[10px] text-slate-400">API {manager?.api_calls ?? 0} • diário {manager?.daily_calls ?? 0}</div>
            <div className="text-[10px] font-black text-slate-400">v{version}</div>
          </div>
        </header>

        <section className="flex-1 overflow-hidden">
          <AnimatePresence mode="wait">
            {view === 'dashboard' && (
              <motion.div key="dashboard" initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="h-full flex flex-col items-center justify-center p-8 text-center">
                <h1 className="text-5xl font-black mb-4">MVP pronto para operação</h1>
                <p className="text-slate-500 mb-8">Gerencie projetos com gestor dedicado e Bíblia viva em docs/.</p>
                <button onClick={() => setView('projects')} className="bg-indigo-600 text-white px-8 py-4 rounded-2xl font-bold">Ir para Projetos</button>
              </motion.div>
            )}

            {view === 'projects' && (
              <motion.div key="projects" initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="h-full p-6 overflow-auto relative">
                {controlling && (
                  <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="absolute inset-0 z-50 bg-white/90 backdrop-blur-md flex flex-col items-center justify-center p-6 text-center">
                    <div className="w-16 h-16 border-4 border-red-500 border-t-transparent rounded-full animate-spin mb-4"></div>
                    <h3 className="text-2xl font-black mb-2 text-red-600 uppercase tracking-tighter">DESTRUIÇÃO EM CURSO...</h3>
                    <p className="text-slate-500 text-sm max-w-sm font-bold">Tarefa irreversível: Removendo Agente e limpando Gateway OpenClaw.</p>
                  </motion.div>
                )}
                {filteredProjects.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-[60vh] text-center p-8 border-2 border-dashed border-slate-200 rounded-3xl bg-slate-50/50">
                    <div className="w-16 h-16 bg-white shadow-sm rounded-2xl flex items-center justify-center mb-4 text-slate-300">
                      <Trash2 size={32} />
                    </div>
                    <h3 className="text-lg font-bold text-slate-900 mb-2">Ambiente Limpo</h3>
                    <p className="text-sm text-slate-500 max-w-xs mb-6">
                      Você não possui projetos cadastrados no momento. Comece criando um novo projeto para gerenciar seus agentes.
                    </p>
                    <button 
                      onClick={() => setView('newProject')}
                      className="bg-indigo-600 hover:bg-indigo-700 text-white px-6 py-3 rounded-xl font-bold text-sm shadow-sm transition-all flex items-center gap-2"
                    >
                      <Plus size={16} /> CRIAR PRIMEIRO PROJETO
                    </button>
                  </div>
                ) : (
                  <>
                    <div className="flex items-center justify-between mb-4 gap-3 flex-wrap">
                      <div className="flex items-center gap-2 flex-wrap">
                        <button onClick={() => setDisplayMode('grid')} className={`px-3 py-2 rounded-lg text-xs font-bold flex items-center gap-1 ${displayMode === 'grid' ? 'bg-indigo-600 text-white' : 'bg-white border border-slate-200'}`}><LayoutGrid size={14} /> Grid</button>
                        <button onClick={() => setDisplayMode('table')} className={`px-3 py-2 rounded-lg text-xs font-bold flex items-center gap-1 ${displayMode === 'table' ? 'bg-indigo-600 text-white' : 'bg-white border border-slate-200'}`}><Table2 size={14} /> Tabela</button>
                        <button onClick={() => setStatusFilter('all')} className={`px-3 py-2 rounded-lg text-xs font-bold ${statusFilter === 'all' ? 'bg-slate-900 text-white' : 'bg-white border border-slate-200 text-slate-600'}`}>Todos</button>
                        <button onClick={() => setStatusFilter('active')} className={`px-3 py-2 rounded-lg text-xs font-bold ${statusFilter === 'active' ? 'bg-emerald-600 text-white' : 'bg-white border border-slate-200 text-slate-600'}`}>Ativos</button>
                        <button onClick={() => setStatusFilter('offline')} className={`px-3 py-2 rounded-lg text-xs font-bold ${statusFilter === 'offline' ? 'bg-amber-500 text-white' : 'bg-white border border-slate-200 text-slate-600'}`}>Offline</button>
                        <button onClick={() => setStatusFilter('paused')} className={`px-3 py-2 rounded-lg text-xs font-bold ${statusFilter === 'paused' ? 'bg-orange-500 text-white' : 'bg-white border border-slate-200 text-slate-600'}`}>Pausado</button>
                      </div>
                      <button onClick={() => setView('newProject')} className="bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-bold flex items-center gap-2 shadow-sm"><Plus size={16} /> Novo Projeto</button>
                    </div>

                    {displayMode === 'grid' ? (
                      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                        {filteredProjects.map((p) => (
                          <div key={p.id} className="group relative bg-white border rounded-2xl p-4 hover:border-indigo-400 transition-all shadow-sm hover:shadow-md flex flex-col justify-between h-[140px]">
                            <button onClick={() => openProject(p.id)} className="text-left flex-1 min-w-0">
                              <div className="font-bold text-sm mb-1 truncate text-indigo-900">{p.name || p.id}</div>
                              <div className="text-xs text-slate-500 mb-2 line-clamp-2">{p.description || 'Sem descrição'}</div>
                              <div className="text-[10px] font-black text-slate-400 uppercase">Status: {p.manager_status}</div>
                            </button>
                            <button 
                              onClick={(e) => { e.stopPropagation(); deleteProject(p.id, p.name); }}
                              className="absolute bottom-4 right-4 p-2 text-slate-300 hover:text-red-500 transition-colors opacity-0 group-hover:opacity-100"
                              title="Deletar Projeto"
                            >
                              <Trash2 size={16} />
                            </button>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="bg-white border rounded-2xl overflow-hidden shadow-sm">
                        <table className="w-full text-sm">
                          <thead className="bg-slate-50"><tr><th className="text-left p-4 text-slate-600 font-black uppercase text-[10px]">Projeto</th><th className="text-left p-4 text-slate-600 font-black uppercase text-[10px]">Descrição</th><th className="text-left p-4 text-slate-600 font-black uppercase text-[10px]">Gestor</th><th className="text-right p-4 text-slate-600 font-black uppercase text-[10px]">Ações</th></tr></thead>
                          <tbody>
                            {filteredProjects.map((p) => (
                              <tr key={p.id} className="border-t group hover:bg-slate-50">
                                <td className="p-4 font-bold text-indigo-900">{p.name || p.id}</td>
                                <td className="p-4 text-slate-500 max-w-xs truncate">{p.description || '-'}</td>
                                <td className="p-4"><span className={`px-2 py-1 rounded-full text-[10px] font-black ${p.manager_status === 'active' ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-600'}`}>{p.manager_status.toUpperCase()}</span></td>
                                <td className="p-4 text-right flex justify-end gap-2">
                                  <button onClick={() => openProject(p.id)} className="text-indigo-600 font-black text-xs hover:underline">ABRIR</button>
                                  <button onClick={() => deleteProject(p.id, p.name)} className="text-slate-300 hover:text-red-500 transition-colors"><Trash2 size={14} /></button>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </>
                )}
              </motion.div>
            )}

            {view === 'newProject' && (
              <motion.div key="newProject" initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="h-full p-6 flex items-start justify-center overflow-auto">
                <div className="w-full max-w-2xl bg-white border rounded-2xl p-8 relative overflow-hidden shadow-xl mb-10">
                  {creatingProject && (
                    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="absolute inset-0 z-50 bg-white/90 backdrop-blur-sm flex flex-col items-center justify-center p-6 text-center">
                      <div className="w-12 h-12 border-4 border-indigo-600 border-t-transparent rounded-full animate-spin mb-4"></div>
                      <h3 className="text-xl font-black mb-2 text-indigo-950">Criando seu Projeto...</h3>
                      <p className="text-slate-500 text-sm max-w-sm">Estamos registrando o subagente, configurando o workspace isolado e preparando a estrutura de documentos.</p>
                      <div className="mt-6 flex gap-1">
                        <span className="w-2 h-2 bg-indigo-600 rounded-full animate-bounce"></span>
                        <span className="w-2 h-2 bg-indigo-600 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></span>
                        <span className="w-2 h-2 bg-indigo-600 rounded-full animate-bounce" style={{ animationDelay: '0.4s' }}></span>
                      </div>
                    </motion.div>
                  )}
                  <h3 className="text-2xl font-black mb-6 text-indigo-900 border-b pb-4">Novo Projeto Agent-Native</h3>
                  
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div className="space-y-4">
                      <div>
                        <label className="text-[10px] uppercase font-black text-slate-500 tracking-wider">Identificação do Projeto</label>
                        <input value={newName} onChange={(e) => setNewName(e.target.value)} className="w-full mt-1 border rounded-xl px-4 py-3 bg-slate-50 focus:bg-white focus:ring-2 focus:ring-indigo-500 outline-none transition-all" placeholder="Nome (ex: watink-saas)" />
                      </div>
                      
                      <div>
                        <label className="text-[10px] uppercase font-black text-slate-500 tracking-wider">Descrição Inicial (Alma do Agente)</label>
                        <textarea value={newDescription} onChange={(e) => setNewDescription(e.target.value)} className="w-full mt-1 border rounded-xl px-4 py-3 bg-slate-50 focus:bg-white focus:ring-2 focus:ring-indigo-500 outline-none transition-all min-h-[120px]" placeholder="O que este projeto faz? Qual o objetivo principal?" />
                      </div>

                      <div>
                        <label className="text-[10px] uppercase font-black text-slate-500 tracking-wider">Vibe do Gestor</label>
                        <select value={vibe} onChange={(e) => setVibe(e.target.value)} className="w-full mt-1 border rounded-xl px-4 py-3 bg-slate-50 outline-none cursor-pointer">
                          <option>Profissional e Pragmático</option>
                          <option>Criativo e Inovador</option>
                          <option>Rígido e Analítico</option>
                          <option>Amigável e Colaborativo</option>
                          <option>Sarcástico e Eficiente</option>
                        </select>
                      </div>
                    </div>

                    <div className="space-y-4 border-l pl-6">
                      <div>
                        <label className="text-[10px] uppercase font-black text-slate-500 tracking-wider">Líder do Projeto (Reporte)</label>
                        <input value={leaderName} onChange={(e) => setLeaderName(e.target.value)} className="w-full mt-1 border rounded-xl px-4 py-3 bg-slate-50 outline-none" placeholder="Nome do Líder" />
                      </div>

                      <div>
                        <label className="text-[10px] uppercase font-black text-slate-500 tracking-wider">E-mail do Líder</label>
                        <input value={leaderEmail} onChange={(e) => setLeaderEmail(e.target.value)} className="w-full mt-1 border rounded-xl px-4 py-3 bg-slate-50 outline-none" placeholder="lider@empresa.com" />
                      </div>

                      <div>
                        <label className="text-[10px] uppercase font-black text-slate-500 tracking-wider">Localização (Cidade/País)</label>
                        <input value={location} onChange={(e) => setLocation(e.target.value)} className="w-full mt-1 border rounded-xl px-4 py-3 bg-slate-50 outline-none" placeholder="São Paulo, Brasil" />
                      </div>

                      <div className="pt-2">
                        <label className="text-[10px] uppercase font-black text-slate-500 tracking-wider block mb-2">Estado do Repositório</label>
                        <div className="flex gap-2 p-1 bg-slate-100 rounded-xl">
                          <button onClick={() => setProjectType('new')} className={`flex-1 py-2 rounded-lg text-xs font-bold transition-all ${projectType === 'new' ? 'bg-white shadow-sm text-indigo-600' : 'text-slate-500'}`}>Projeto Novo</button>
                          <button onClick={() => setProjectType('existing')} className={`flex-1 py-2 rounded-lg text-xs font-bold transition-all ${projectType === 'existing' ? 'bg-white shadow-sm text-indigo-600' : 'text-slate-500'}`}>Existente</button>
                        </div>
                      </div>

                      {projectType === 'existing' && (
                        <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }}>
                          <label className="text-[10px] uppercase font-black text-slate-500 tracking-wider">URL do Repositório Git</label>
                          <input value={gitUrl} onChange={(e) => setGitUrl(e.target.value)} className="w-full mt-1 border rounded-xl px-4 py-3 bg-slate-50 outline-none border-indigo-200" placeholder="https://github.com/..." />
                        </motion.div>
                      )}
                    </div>
                  </div>

                  <div className="mt-8 pt-6 border-t flex justify-end">
                    <button onClick={createProject} disabled={creatingProject || !newName.trim()} className="bg-indigo-600 text-white px-10 py-4 rounded-2xl font-black apple-shadow hover:bg-indigo-700 disabled:opacity-60 transition-all flex items-center gap-3">
                      {creatingProject ? 'CONSTRUINDO...' : 'CRIAR E INICIAR GESTÃO'}
                      <Bot size={20} />
                    </button>
                  </div>
                </div>
              </motion.div>
            )}

            {view === 'chat' && (
              <motion.div key="chat" initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="h-full p-6 relative overflow-hidden">
                {controlling && (
                  <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="absolute inset-0 z-50 bg-white/80 backdrop-blur-md flex flex-col items-center justify-center p-6 text-center">
                    <div className="w-16 h-16 border-4 border-indigo-600 border-t-transparent rounded-full animate-spin mb-4"></div>
                    <h3 className="text-2xl font-black mb-2 text-indigo-950">Ativando o Motor de Gestão...</h3>
                    <p className="text-slate-500 text-sm max-w-sm">Estamos enviando a diretriz master e sincronizando o Roadmap com o subagente.</p>
                  </motion.div>
                )}
                <div className="h-full flex gap-4">
                  <div className="flex-1 flex flex-col min-w-0">
                    <div className="mb-3 text-xs text-slate-500 font-semibold">Projeto: {selectedProject?.name || selectedProject?.id || 'não selecionado'} • Sessão: {manager?.manager_session_key || selectedProject?.manager_session_key || '-'}</div>
                    <div className="mb-3 flex gap-2">
                      <button onClick={() => handleControl('restart')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-slate-900 text-white disabled:opacity-50">Reiniciar</button>
                      <button onClick={() => handleControl('pause')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-amber-500 text-white disabled:opacity-50">Pausar</button>
                      <button onClick={() => handleControl('resume')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-emerald-600 text-white disabled:opacity-50">Retomar</button>
                    </div>
                    <div className="flex-1 space-y-4 overflow-y-auto mb-4 pr-2">
                      {chatHistory.map((item, idx) => <ChatBubble key={idx} sender={item.sender} message={item.message} />)}
                      {sending && (
                        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex justify-start items-end space-x-2">
                          <div className="w-7 h-7 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-600 mb-1 flex-shrink-0"><Bot size={14} /></div>
                          <div className="max-w-[85%] p-3 rounded-2xl bg-white text-gray-500 rounded-bl-none border border-gray-100 text-sm">
                            <span className="inline-flex items-center gap-2">
                              <span className="w-2 h-2 bg-indigo-500 rounded-full animate-pulse"></span>
                              pensando...
                            </span>
                          </div>
                        </motion.div>
                      )}
                    </div>
                    <div className="relative">
                      <input type="text" value={message} onChange={(e) => setMessage(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()} placeholder="Converse com o gestor..." className="w-full bg-white border border-gray-200 rounded-[22px] py-4 px-6 pr-14 focus:outline-none focus:border-indigo-500" />
                      <button onClick={handleSendMessage} disabled={sending} className="absolute right-3 top-1/2 -translate-y-1/2 w-10 h-10 bg-indigo-600 rounded-full flex items-center justify-center text-white disabled:opacity-60"><Send size={18} /></button>
                    </div>
                  </div>

                  <aside className="hidden lg:flex w-[360px] flex-col gap-3 overflow-y-auto">
                    <Panel title="Progresso da Triagem">
                      <div className="flex gap-1 flex-wrap">{plannerStages.map((stg, idx) => <span key={stg} className={`px-2 py-1 rounded text-[10px] font-bold ${idx <= stageIndex ? 'bg-indigo-600 text-white' : 'bg-slate-100 text-slate-500'}`}>{idx + 1}. {stageLabels[stg]}</span>)}</div>
                      <div className="text-xs text-slate-600 mt-2">Atual: <b>{planner ? (stageLabels[planner.stage] || planner.stage) : 'Tipo'}</b></div>
                    </Panel>
                    <Panel title="Entregáveis (docs/)">{(planner?.deliverables || []).map((d) => { const done = (planner?.deliverables_done || []).includes(d); return <div key={d} className={`text-xs font-semibold ${done ? 'text-emerald-600' : 'text-slate-400'}`}>{done ? '✅' : '⏳'} {d}</div>; })}</Panel>
                    <Panel title="Checkpoint Git"><div className="text-xs break-all">{planner?.last_checkpoint || 'Sem checkpoint ainda.'}</div></Panel>
                    <Panel title="Aprendizado do gestor">
                      <div className="text-xs text-slate-600 mb-3">O gestor aprende automaticamente com conversa + docs (PLANNING e entregáveis).</div>
                      {planner?.stage === 'active' && (
                        <button 
                          onClick={() => handleControl('start-execution')}
                          disabled={controlling}
                          className="w-full py-3 px-4 bg-indigo-600 text-white rounded-xl font-black text-xs hover:bg-indigo-700 transition-colors apple-shadow disabled:opacity-50"
                        >
                          🚀 START PROJETO (ATIVAR BÍBLIA)
                        </button>
                      )}
                    </Panel>
                  </aside>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </section>
      </main>
    </div>
  );
};

const Panel = ({ title, children }: { title: string; children: React.ReactNode }) => (
  <div className="p-3 rounded-xl bg-white border border-slate-200"><div className="text-[10px] font-black uppercase tracking-widest text-slate-500 mb-2">{title}</div>{children}</div>
);

const NavItem = ({ icon, label, active = false, onClick }: NavItemProps) => (
  <div onClick={onClick} className={`flex items-center space-x-3 px-4 py-3 rounded-2xl cursor-pointer transition-all duration-200 ${active ? 'bg-indigo-600 text-white apple-shadow' : 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'}`}>{icon}<span className="font-bold text-sm hidden md:block">{label}</span></div>
);

const ChatBubble = ({ sender, message }: { sender: 'agent' | 'user'; message: string }) => (
  <motion.div initial={{ opacity: 0, x: sender === 'user' ? 20 : -20 }} animate={{ opacity: 1, x: 0 }} className={`flex ${sender === 'user' ? 'justify-end' : 'justify-start'} items-end space-x-2`}>
    {sender === 'agent' && <div className="w-7 h-7 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-600 mb-1 flex-shrink-0"><Bot size={14} /></div>}
    <div className={`max-w-[85%] p-4 rounded-2xl text-sm apple-shadow markdown-body ${sender === 'user' ? 'bg-indigo-600 text-white rounded-br-none' : 'bg-white text-gray-900 rounded-bl-none border border-gray-100'}`}>
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{message}</ReactMarkdown>
    </div>
  </motion.div>
);

export default Dashboard;
