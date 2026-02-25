import { motion, AnimatePresence } from 'framer-motion';
import { useEffect, useRef, useState } from 'react';
import { LayoutDashboard, MessageSquare, Settings, Activity, Plus, Send, ArrowLeft, Bot, FileText, Target, CheckCircle2 } from 'lucide-react';

interface NavItemProps {
  icon: React.ReactNode;
  label: string;
  active?: boolean;
  onClick?: () => void;
}

type ChatItem = { sender: 'agent' | 'user'; message: string };
type Project = { id: string; name: string; manager_status: string; manager_session_key: string };
type ManagerInfo = { manager_status: string; manager_enabled: boolean; manager_session_key: string; api_calls: number };
type ProjectMessage = { id: string; sender: 'agent' | 'user'; message: string };

const Dashboard = () => {
  const [view, setView] = useState<'empty' | 'chat'>('empty');
  const [message, setMessage] = useState('');
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<string>('');
  const [manager, setManager] = useState<ManagerInfo | null>(null);
  const [creatingProject, setCreatingProject] = useState(false);
  const [sending, setSending] = useState(false);
  const [controlling, setControlling] = useState(false);
  const [projectSummary, setProjectSummary] = useState<string>('');

  const [chatHistory, setChatHistory] = useState<ChatItem[]>([
    { sender: 'agent', message: 'Olá! Selecione/crie um projeto e converse com o gestor dedicado.' },
  ]);

  const version = '0.1.6-beta';
  const reconnectAttemptedRef = useRef<Record<string, boolean>>({});
  const selectedProject = projects.find((p) => p.id === selectedProjectId) || null;

  const loadProjects = async () => {
    try {
      const res = await fetch('/api/projects');
      if (!res.ok) return;
      const data: Project[] = await res.json();
      setProjects(data);
      if (!selectedProjectId && data.length > 0) {
        setSelectedProjectId(data[0].id);
      }
    } catch {
      // noop
    }
  };

  const loadManager = async (projectId: string) => {
    try {
      const res = await fetch(`/api/projects/${projectId}/manager`);
      if (!res.ok) return null;
      const data = await res.json();
      setManager(data);
      return data as ManagerInfo;
    } catch {
      return null;
    }
  };

  const loadSummary = async (projectId: string) => {
    try {
      const res = await fetch(`/api/projects/${projectId}/summary`);
      if (!res.ok) return;
      const data = await res.json();
      setProjectSummary(data.summary || '');
    } catch {
      // noop
    }
  };

  const loadMessages = async (projectId: string) => {
    try {
      const res = await fetch(`/api/projects/${projectId}/messages`);
      if (!res.ok) return;
      const data: ProjectMessage[] = await res.json();
      if (data.length > 0) {
        setChatHistory(data.map((m) => ({ sender: m.sender, message: m.message })));
      } else {
        setChatHistory([{ sender: 'agent', message: 'Sem histórico ainda. Envie a primeira mensagem para o gestor.' }]);
      }
    } catch {
      // noop
    }
  };

  useEffect(() => {
    loadProjects();
  }, []);

  useEffect(() => {
    const bootProject = async () => {
      if (!selectedProjectId) return;
      await loadMessages(selectedProjectId);
      await loadSummary(selectedProjectId);
      const info = await loadManager(selectedProjectId);

      const needReconnect = info && (info.manager_status === 'offline' || info.manager_status === 'paused');
      if (needReconnect && !reconnectAttemptedRef.current[selectedProjectId]) {
        reconnectAttemptedRef.current[selectedProjectId] = true;
        await fetch(`/api/projects/${selectedProjectId}/manager/control`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: 'resume' }),
        });
        await loadManager(selectedProjectId);
        setChatHistory((prev) => [...prev, { sender: 'agent', message: 'Auto-reconnect aplicado no gestor deste projeto.' }]);
      }
    };
    bootProject();
  }, [selectedProjectId]);

  const handleStartProject = async () => {
    setView('chat');
    if (creatingProject) return;

    setCreatingProject(true);
    try {
      const payload = {
        name: `projeto-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}`,
        description: 'Projeto iniciado via dashboard ClawProject',
      };
      const res = await fetch('/api/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error();
      const created: Project = await res.json();
      await loadProjects();
      setSelectedProjectId(created.id);
      await loadManager(created.id);
      await loadSummary(created.id);
      setChatHistory((prev) => [...prev, { sender: 'agent', message: `Projeto criado. Sessão do gestor: ${created.manager_session_key}` }]);
    } catch {
      setChatHistory((prev) => [...prev, { sender: 'agent', message: 'Falha ao criar projeto agora.' }]);
    } finally {
      setCreatingProject(false);
    }
  };

  const handleControl = async (action: 'restart' | 'pause' | 'resume') => {
    if (!selectedProjectId || controlling) return;
    setControlling(true);
    try {
      const res = await fetch(`/api/projects/${selectedProjectId}/manager/control`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error();
      setChatHistory((prev) => [...prev, { sender: 'agent', message: `Gestor: ação '${action}' aplicada. Status: ${data.manager_status}` }]);
      await loadProjects();
      await loadManager(selectedProjectId);
      await loadSummary(selectedProjectId);
    } catch {
      setChatHistory((prev) => [...prev, { sender: 'agent', message: `Falha ao executar '${action}' no gestor.` }]);
    } finally {
      setControlling(false);
    }
  };

  const handleSendMessage = async () => {
    if (!message.trim() || sending) return;
    const userText = message;
    setMessage('');
    setChatHistory((prev) => [...prev, { sender: 'user', message: userText }]);

    if (!selectedProjectId) {
      setChatHistory((prev) => [...prev, { sender: 'agent', message: 'Selecione um projeto antes de enviar mensagens.' }]);
      return;
    }

    setSending(true);
    try {
      const res = await fetch(`/api/projects/${selectedProjectId}/manager/message`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: userText }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error();
      await loadMessages(selectedProjectId);
      if (!data?.reply) {
        setChatHistory((prev) => [...prev, { sender: 'agent', message: 'Recebido.' }]);
      }
      await loadManager(selectedProjectId);
      await loadSummary(selectedProjectId);
    } catch {
      setChatHistory((prev) => [...prev, { sender: 'agent', message: 'Gestor indisponível no momento.' }]);
    } finally {
      setSending(false);
    }
  };

  const managerOnline = manager?.manager_status === 'active';

  return (
    <div className="flex h-screen w-full bg-[#F5F5F7] overflow-hidden text-[#1D1D1F]">
      <aside className="w-20 md:w-72 glass border-r border-gray-200 flex flex-col p-4 z-20">
        <div className="flex items-center space-x-3 px-2 mb-6 mt-2">
          <div className="w-10 h-10 bg-indigo-600 rounded-xl flex items-center justify-center text-white font-black text-xl">C</div>
          <span className="font-extrabold text-xl tracking-tighter hidden md:block">ClawProject</span>
        </div>

        <nav className="space-y-2 mb-4">
          <NavItem icon={<LayoutDashboard size={20} />} label="Dashboard" active={view === 'empty'} onClick={() => setView('empty')} />
          <NavItem icon={<MessageSquare size={20} />} label="Projetos" active={view === 'chat'} onClick={() => setView('chat')} />
          <NavItem icon={<Activity size={20} />} label="Atividade" />
        </nav>

        <div className="hidden md:block">
          <div className="text-[10px] font-black uppercase text-slate-400 mb-2 px-2">Projetos</div>
          <div className="space-y-2 max-h-[280px] overflow-y-auto pr-1">
            {projects.map((p) => (
              <button
                key={p.id}
                onClick={() => {
                  setSelectedProjectId(p.id);
                  setView('chat');
                }}
                className={`w-full text-left px-3 py-2 rounded-xl border ${selectedProjectId === p.id ? 'border-indigo-400 bg-indigo-50' : 'border-gray-200 bg-white'}`}
              >
                <div className="text-xs font-bold text-slate-800 truncate">{p.name || p.id}</div>
                <div className="text-[10px] text-slate-500">{p.manager_status}</div>
              </button>
            ))}
          </div>
        </div>

        <div className="pt-4 border-t border-gray-100 mt-auto">
          <NavItem icon={<Settings size={20} />} label="Configurações" />
        </div>
      </aside>

      <main className="flex-1 flex flex-col relative overflow-hidden">
        <header className="h-20 glass border-b border-gray-200 flex items-center justify-between px-8 z-10">
          <div className="flex items-center space-x-4">
            {view === 'chat' && <button onClick={() => setView('empty')} className="p-2 hover:bg-gray-100 rounded-full"><ArrowLeft size={20} /></button>}
            <h2 className="text-2xl font-bold tracking-tight text-gray-900">{view === 'empty' ? 'Visão Geral' : 'Modo Planejador'}</h2>
          </div>

          <div className="flex items-center gap-5">
            <div className="text-right">
              <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Gestor do Projeto</p>
              <div className="flex items-center justify-end gap-2">
                <span className={`h-2.5 w-2.5 rounded-full ${managerOnline ? 'bg-emerald-500' : 'bg-amber-500'}`} />
                <span className={`text-xs font-bold uppercase ${managerOnline ? 'text-emerald-600' : 'text-amber-600'}`}>{managerOnline ? 'Online' : 'Offline'}</span>
              </div>
              <p className="text-[10px] text-slate-400">API calls: {manager?.api_calls ?? 0}</p>
            </div>
            <div className="flex flex-col items-end">
              <span className="text-sm font-bold text-emerald-600 uppercase tracking-widest">Gateway Ativo</span>
              <span className="text-[10px] font-black text-slate-400 uppercase tracking-tighter">v{version}</span>
            </div>
          </div>
        </header>

        <section className="flex-1 relative overflow-hidden">
          <AnimatePresence mode="wait">
            {view === 'empty' ? (
              <motion.div key="empty" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="h-full flex flex-col items-center justify-center text-center max-w-4xl mx-auto p-8">
                <div className="w-24 h-24 bg-white rounded-[32px] apple-shadow-lg flex items-center justify-center mb-8 border border-gray-100"><Target size={40} className="text-indigo-600" /></div>
                <h1 className="text-5xl font-black text-gray-900 mb-4 tracking-tighter">Seu próximo passo começa aqui.</h1>
                <p className="text-gray-500 text-xl mb-10 leading-relaxed font-medium max-w-2xl">Crie projetos com gestor dedicado, controles operacionais e entregáveis em docs/.</p>
                <button onClick={handleStartProject} disabled={creatingProject} className="bg-indigo-600 hover:bg-indigo-700 text-white px-10 py-5 rounded-[28px] font-bold text-xl flex items-center space-x-3 mb-16 disabled:opacity-70">
                  <Plus size={28} />
                  <span>{creatingProject ? 'Criando projeto...' : 'Criar Novo Projeto'}</span>
                </button>
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 w-full">
                  <DeliverableCard icon={<Bot size={18} />} title="Software" desc="PRD + DER + POPs" />
                  <DeliverableCard icon={<Target size={18} />} title="Vendas" desc="Funil + Scripts" />
                  <DeliverableCard icon={<FileText size={18} />} title="Conteúdo" desc="Calendário + Estilo" />
                  <DeliverableCard icon={<CheckCircle2 size={18} />} title="Gestão" desc="Checklists + Ops" />
                </div>
              </motion.div>
            ) : (
              <motion.div key="chat" initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }} className="h-full flex flex-col p-8 max-w-4xl mx-auto">
                <div className="mb-3 text-xs text-slate-500 font-semibold">
                  Projeto: {selectedProject?.name || selectedProject?.id || 'não selecionado'} • Sessão gestor: {manager?.manager_session_key || selectedProject?.manager_session_key || '-'}
                </div>
                <div className="mb-4 p-3 rounded-xl bg-slate-50 border border-slate-200">
                  <div className="text-[10px] font-black uppercase tracking-widest text-slate-500 mb-1">Contexto sumarizado do projeto</div>
                  <div className="text-xs text-slate-700 whitespace-pre-wrap">{projectSummary || 'Sem resumo ainda.'}</div>
                </div>

                <div className="mb-4 flex gap-2">
                  <button onClick={() => handleControl('restart')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-slate-900 text-white disabled:opacity-50">Reiniciar Gestor</button>
                  <button onClick={() => handleControl('pause')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-amber-500 text-white disabled:opacity-50">Pausar</button>
                  <button onClick={() => handleControl('resume')} disabled={!selectedProjectId || controlling} className="px-3 py-2 text-xs font-bold rounded-lg bg-emerald-600 text-white disabled:opacity-50">Retomar</button>
                </div>

                <div className="flex-1 space-y-6 overflow-y-auto mb-6 pr-4">
                  {chatHistory.map((item, idx) => <ChatBubble key={idx} sender={item.sender} message={item.message} />)}
                </div>

                <div className="relative group">
                  <input type="text" value={message} onChange={(e) => setMessage(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()} placeholder="Converse com o gestor dedicado do projeto..." className="w-full bg-white border border-gray-200 rounded-[28px] py-5 px-8 pr-16 apple-shadow-lg focus:outline-none focus:border-indigo-500 text-lg font-medium" />
                  <button onClick={handleSendMessage} disabled={sending} className="absolute right-4 top-1/2 -translate-y-1/2 w-12 h-12 bg-indigo-600 rounded-full flex items-center justify-center text-white disabled:opacity-60"><Send size={20} /></button>
                </div>
                <p className="text-center text-[10px] text-gray-400 font-black uppercase tracking-widest mt-4">Bíblia do Projeto em docs/PLANNING.md</p>
              </motion.div>
            )}
          </AnimatePresence>
        </section>
      </main>
    </div>
  );
};

const NavItem = ({ icon, label, active = false, onClick }: NavItemProps) => (
  <div onClick={onClick} className={`flex items-center space-x-3 px-4 py-3 rounded-2xl cursor-pointer transition-all duration-200 ${active ? 'bg-indigo-600 text-white apple-shadow' : 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'}`}>
    {icon}
    <span className="font-bold text-sm hidden md:block">{label}</span>
  </div>
);

const ChatBubble = ({ sender, message }: { sender: 'agent' | 'user'; message: string }) => (
  <motion.div initial={{ opacity: 0, x: sender === 'user' ? 20 : -20 }} animate={{ opacity: 1, x: 0 }} className={`flex ${sender === 'user' ? 'justify-end' : 'justify-start'} items-end space-x-3`}>
    {sender === 'agent' && <div className="w-8 h-8 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-600 mb-1"><Bot size={18} /></div>}
    <div className={`max-w-[80%] p-6 rounded-[28px] text-lg font-medium apple-shadow ${sender === 'user' ? 'bg-indigo-600 text-white rounded-br-none' : 'bg-white text-gray-900 rounded-bl-none border border-gray-50'}`}>{message}</div>
  </motion.div>
);

const DeliverableCard = ({ icon, title, desc }: { icon: React.ReactNode; title: string; desc: string }) => (
  <div className="bg-white p-4 rounded-[20px] border border-gray-100 text-center hover:border-indigo-200 transition-all cursor-default apple-shadow group">
    <div className="w-10 h-10 bg-slate-50 rounded-xl flex items-center justify-center mx-auto mb-3 text-slate-400 group-hover:text-indigo-600 group-hover:bg-indigo-50 transition-colors">{icon}</div>
    <h4 className="font-bold text-gray-900 text-sm">{title}</h4>
    <p className="text-[10px] text-gray-400 font-black uppercase tracking-tighter mt-1">{desc}</p>
  </div>
);

export default Dashboard;
