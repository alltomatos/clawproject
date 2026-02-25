import { motion, AnimatePresence } from 'framer-motion';
import { useEffect, useRef, useState } from 'react';
import { LayoutDashboard, MessageSquare, Settings, Activity, Plus, Send, ArrowLeft, Bot, Target } from 'lucide-react';

interface NavItemProps { icon: React.ReactNode; label: string; active?: boolean; onClick?: () => void; }
type ChatItem = { sender: 'agent' | 'user'; message: string };
type Project = { id: string; name: string; manager_status: string; manager_session_key: string };
type ManagerInfo = { manager_status: string; manager_enabled: boolean; manager_session_key: string; api_calls: number; daily_calls?: number };
type ProjectMessage = { id: string; sender: 'agent' | 'user'; message: string };
type PlannerInfo = { stage: string; project_type: string; niche: string; deliverables: string[]; deliverables_done: string[]; last_checkpoint: string };

const Dashboard = () => {
  const [view, setView] = useState<'empty' | 'chat'>('empty');
  const [message, setMessage] = useState('');
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<string>('');
  const [manager, setManager] = useState<ManagerInfo | null>(null);
  const [planner, setPlanner] = useState<PlannerInfo | null>(null);
  const [creatingProject, setCreatingProject] = useState(false);
  const [sending, setSending] = useState(false);
  const [controlling, setControlling] = useState(false);
  const [chatHistory, setChatHistory] = useState<ChatItem[]>([{ sender: 'agent', message: 'Selecione/crie um projeto e converse com o gestor.' }]);
  const version = '0.1.8-beta';
  const reconnectAttemptedRef = useRef<Record<string, boolean>>({});

  const selectedProject = projects.find((p) => p.id === selectedProjectId) || null;
  const plannerStages = ['triage_type', 'triage_niche', 'objective', 'deliverables', 'active'];
  const stageLabels: Record<string, string> = { triage_type: 'Tipo', triage_niche: 'Nicho', objective: 'Objetivo', deliverables: 'Entregáveis', active: 'Execução' };
  const stageIndex = planner ? Math.max(0, plannerStages.indexOf(planner.stage)) : 0;
  const managerOnline = manager?.manager_status === 'active';

  const loadProjects = async () => {
    const res = await fetch('/api/projects'); if (!res.ok) return;
    const data: Project[] = await res.json(); setProjects(data);
    if (!selectedProjectId && data.length > 0) setSelectedProjectId(data[0].id);
  };
  const loadManager = async (projectId: string) => { const r = await fetch(`/api/projects/${projectId}/manager`); if (!r.ok) return; setManager(await r.json()); };
  const loadPlanner = async (projectId: string) => { const r = await fetch(`/api/projects/${projectId}/planner`); if (!r.ok) return; setPlanner(await r.json()); };
  const loadMessages = async (projectId: string) => {
    const r = await fetch(`/api/projects/${projectId}/messages`); if (!r.ok) return;
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
        await fetch(`/api/projects/${selectedProjectId}/manager/control`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action: 'resume' }) });
        await loadManager(selectedProjectId);
      }
    };
    boot();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedProjectId]);

  const handleStartProject = async () => {
    setView('chat'); if (creatingProject) return; setCreatingProject(true);
    try {
      const payload = { name: `projeto-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}`, description: 'Projeto iniciado via dashboard ClawProject' };
      const r = await fetch('/api/projects', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
      if (!r.ok) throw new Error();
      const created: Project = await r.json();
      await loadProjects(); setSelectedProjectId(created.id); setChatHistory((prev) => [...prev, { sender: 'agent', message: `Projeto criado. Sessão gestor: ${created.manager_session_key}` }]);
    } catch { setChatHistory((prev) => [...prev, { sender: 'agent', message: 'Falha ao criar projeto.' }]); }
    finally { setCreatingProject(false); }
  };

  const handleSendMessage = async () => {
    if (!message.trim() || sending || !selectedProjectId) return;
    const userText = message; setMessage(''); setChatHistory((p) => [...p, { sender: 'user', message: userText }]); setSending(true);
    try {
      const r = await fetch(`/api/projects/${selectedProjectId}/manager/message`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ message: userText }) });
      if (!r.ok) throw new Error();
      await Promise.all([loadMessages(selectedProjectId), loadManager(selectedProjectId), loadPlanner(selectedProjectId)]);
    } catch { setChatHistory((p) => [...p, { sender: 'agent', message: 'Gestor indisponível no momento.' }]); }
    finally { setSending(false); }
  };

  const handleControl = async (action: 'restart' | 'pause' | 'resume') => {
    if (!selectedProjectId || controlling) return; setControlling(true);
    try {
      await fetch(`/api/projects/${selectedProjectId}/manager/control`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action }) });
      await Promise.all([loadProjects(), loadManager(selectedProjectId), loadPlanner(selectedProjectId)]);
    } finally { setControlling(false); }
  };

  
  return (
    <div className="flex h-screen w-full bg-[#F5F5F7] overflow-hidden text-[#1D1D1F]">
      <aside className="w-20 md:w-72 glass border-r border-gray-200 flex flex-col p-4 z-20">
        <div className="flex items-center space-x-3 px-2 mb-6 mt-2"><div className="w-10 h-10 bg-indigo-600 rounded-xl flex items-center justify-center text-white font-black text-xl">C</div><span className="font-extrabold text-xl tracking-tighter hidden md:block">ClawProject</span></div>
        <nav className="space-y-2 mb-4"><NavItem icon={<LayoutDashboard size={20} />} label="Dashboard" active={view === 'empty'} onClick={() => setView('empty')} /><NavItem icon={<MessageSquare size={20} />} label="Projetos" active={view === 'chat'} onClick={() => setView('chat')} /><NavItem icon={<Activity size={20} />} label="Atividade" /></nav>
        <div className="hidden md:block"><div className="text-[10px] font-black uppercase text-slate-400 mb-2 px-2">Projetos</div><div className="space-y-2 max-h-[280px] overflow-y-auto pr-1">{projects.map((p) => <button key={p.id} onClick={() => { setSelectedProjectId(p.id); setView('chat'); }} className={`w-full text-left px-3 py-2 rounded-xl border ${selectedProjectId === p.id ? 'border-indigo-400 bg-indigo-50' : 'border-gray-200 bg-white'}`}><div className="text-xs font-bold text-slate-800 truncate">{p.name || p.id}</div><div className="text-[10px] text-slate-500">{p.manager_status}</div></button>)}</div></div>
        <div className="pt-4 border-t border-gray-100 mt-auto"><NavItem icon={<Settings size={20} />} label="Configurações" /></div>
      </aside>

      <main className="flex-1 flex flex-col overflow-hidden">
        <header className="h-20 glass border-b border-gray-200 flex items-center justify-between px-8">
          <div className="flex items-center space-x-4">{view === 'chat' && <button onClick={() => setView('empty')} className="p-2 hover:bg-gray-100 rounded-full"><ArrowLeft size={20} /></button>}<h2 className="text-2xl font-bold tracking-tight text-gray-900">{view === 'empty' ? 'Visão Geral' : 'Modo Planejador'}</h2></div>
          <div className="text-right"><div className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Gestor</div><div className={`text-xs font-bold ${managerOnline ? 'text-emerald-600' : 'text-amber-600'}`}>{managerOnline ? 'ONLINE' : 'OFFLINE'}</div><div className="text-[10px] text-slate-400">API {manager?.api_calls ?? 0} • diário {manager?.daily_calls ?? 0}</div><div className="text-[10px] font-black text-slate-400">v{version}</div></div>
        </header>

        <section className="flex-1 overflow-hidden">
          <AnimatePresence mode="wait">
            {view === 'empty' ? (
              <motion.div key="empty" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="h-full flex flex-col items-center justify-center text-center p-8">
                <div className="w-24 h-24 bg-white rounded-[32px] apple-shadow-lg flex items-center justify-center mb-8 border border-gray-100"><Target size={40} className="text-indigo-600" /></div>
                <h1 className="text-5xl font-black text-gray-900 mb-4 tracking-tighter">Seu próximo passo começa aqui.</h1>
                <button onClick={handleStartProject} disabled={creatingProject} className="bg-indigo-600 hover:bg-indigo-700 text-white px-10 py-5 rounded-[28px] font-bold text-xl flex items-center space-x-3 disabled:opacity-70"><Plus size={28} /><span>{creatingProject ? 'Criando...' : 'Criar Novo Projeto'}</span></button>
              </motion.div>
            ) : (
              <motion.div key="chat" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="h-full p-6">
                <div className="h-full flex gap-4">
                  <div className="flex-1 flex flex-col min-w-0">
                    <div className="mb-3 text-xs text-slate-500 font-semibold">Projeto: {selectedProject?.name || selectedProject?.id || 'não selecionado'} • Sessão: {manager?.manager_session_key || selectedProject?.manager_session_key || '-'}</div>
                    <div className="mb-3 flex gap-2"><button onClick={() => handleControl('restart')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-slate-900 text-white disabled:opacity-50">Reiniciar</button><button onClick={() => handleControl('pause')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-amber-500 text-white disabled:opacity-50">Pausar</button><button onClick={() => handleControl('resume')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-emerald-600 text-white disabled:opacity-50">Retomar</button></div>
                    <div className="flex-1 space-y-4 overflow-y-auto mb-4 pr-2">{chatHistory.map((item, idx) => <ChatBubble key={idx} sender={item.sender} message={item.message} />)}</div>
                    <div className="relative"><input type="text" value={message} onChange={(e) => setMessage(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()} placeholder="Converse com o gestor... (você está treinando o agente no contexto do projeto)" className="w-full bg-white border border-gray-200 rounded-[22px] py-4 px-6 pr-14 focus:outline-none focus:border-indigo-500" /><button onClick={handleSendMessage} disabled={sending} className="absolute right-3 top-1/2 -translate-y-1/2 w-10 h-10 bg-indigo-600 rounded-full flex items-center justify-center text-white disabled:opacity-60"><Send size={18} /></button></div>
                  </div>

                  <aside className="hidden lg:flex w-[360px] flex-col gap-3 overflow-y-auto">
                    <Panel title="Progresso da Triagem">
                      <div className="flex gap-1 flex-wrap">{plannerStages.map((stg, idx) => <span key={stg} className={`px-2 py-1 rounded text-[10px] font-bold ${idx <= stageIndex ? 'bg-indigo-600 text-white' : 'bg-slate-100 text-slate-500'}`}>{idx + 1}. {stageLabels[stg]}</span>)}</div>
                      <div className="text-xs text-slate-600 mt-2">Atual: <b>{planner ? (stageLabels[planner.stage] || planner.stage) : 'Tipo'}</b></div>
                    </Panel>
                    <Panel title="Entregáveis (docs/)">{(planner?.deliverables || []).map((d) => { const done = (planner?.deliverables_done || []).includes(d); return <div key={d} className={`text-xs font-semibold ${done ? 'text-emerald-600' : 'text-slate-400'}`}>{done ? '✅' : '⏳'} {d}</div>; })}</Panel>
                    <Panel title="Checkpoint Git"><div className="text-xs break-all">{planner?.last_checkpoint || 'Sem checkpoint ainda.'}</div></Panel>
                    <Panel title="Resumo do Projeto"><div className="text-xs whitespace-pre-wrap text-slate-700">{(chatHistory.find((m) => m.sender === 'agent')?.message || 'Sem resumo ainda.')}</div></Panel>
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
    {sender === 'agent' && <div className="w-7 h-7 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-600 mb-1"><Bot size={14} /></div>}
    <div className={`max-w-[80%] p-4 rounded-2xl text-sm font-medium apple-shadow ${sender === 'user' ? 'bg-indigo-600 text-white rounded-br-none' : 'bg-white text-gray-900 rounded-bl-none border border-gray-100'}`}>{message}</div>
  </motion.div>
);

export default Dashboard;


