import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { LayoutDashboard, MessageSquare, Settings, Activity, Plus, Send, ArrowLeft, Bot } from 'lucide-react';

interface NavItemProps {
  icon: React.ReactNode;
  label: string;
  active?: boolean;
  onClick?: () => void;
}

const Dashboard = () => {
  const [view, setView] = useState<'empty' | 'chat'>('empty');
  const [message, setMessage] = useState('');
  const [chatHistory, setChatHistory] = useState([
    { sender: 'agent', message: 'Olá, Ronaldo! Sou o watinker_bot (Sessão: Main). Estou pronto para planejar seu novo projeto. Para começarmos, qual o nome do projeto e qual o objetivo central?' }
  ]);
  const version = "0.1.2-beta";

  const handleStartProject = () => {
    setView('chat');
  };

  const handleSendMessage = () => {
    if (!message.trim()) return;
    
    // Simulação de envio enquanto estabilizamos o backend
    setChatHistory([...chatHistory, { sender: 'user', message: message }]);
    setMessage('');
    
    // TODO: Chamar API real do ClawFlow que roteia para o OpenClaw
  };

  return (
    <div className="flex h-screen w-full bg-[#F5F5F7] overflow-hidden text-[#1D1D1F]">
      {/* Sidebar - Apple Style */}
      <aside className="w-20 md:w-64 glass border-r border-gray-200 flex flex-col p-4 z-20">
        <div className="flex items-center space-x-3 px-2 mb-10 mt-2">
          <div className="w-10 h-10 bg-indigo-600 rounded-xl flex items-center justify-center text-white font-black text-xl shadow-lg shadow-indigo-200">
            C
          </div>
          <span className="font-extrabold text-xl tracking-tighter hidden md:block">ClawFlow</span>
        </div>

        <nav className="flex-1 space-y-2">
          <NavItem icon={<LayoutDashboard size={20} />} label="Dashboard" active={view === 'empty'} onClick={() => setView('empty')} />
          <NavItem icon={<MessageSquare size={20} />} label="Projetos" active={view === 'chat'} onClick={() => setView('chat')} />
          <NavItem icon={<Activity size={20} />} label="Logs do Agente" />
        </nav>

        <div className="pt-4 border-t border-gray-100">
          <NavItem icon={<Settings size={20} />} label="Configurações" />
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col relative overflow-hidden">
        {/* Header */}
        <header className="h-20 glass border-b border-gray-200 flex items-center justify-between px-8 z-10">
          <div className="flex items-center space-x-4">
            {view === 'chat' && (
              <button onClick={() => setView('empty')} className="p-2 hover:bg-gray-100 rounded-full transition-colors">
                <ArrowLeft size={20} />
              </button>
            )}
            <h2 className="text-2xl font-bold tracking-tight text-gray-900">
              {view === 'empty' ? 'Visão Geral' : 'Modo Planejador'}
            </h2>
          </div>
          
          <div className="flex items-center space-x-6">
            <div className="flex flex-col items-end">
              <div className="flex items-center space-x-2">
                <span className="relative flex h-3 w-3">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
                </span>
                <span className="text-sm font-bold text-emerald-600 uppercase tracking-widest">Gateway Online</span>
              </div>
              <span className="text-[10px] font-black text-slate-400 uppercase tracking-tighter mt-0.5">v{version}</span>
            </div>
          </div>
        </header>

        {/* Content Area with AnimatePresence */}
        <section className="flex-1 relative overflow-hidden">
          <AnimatePresence mode="wait">
            {view === 'empty' ? (
              <motion.div 
                key="empty"
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 1.05 }}
                transition={{ duration: 0.4, ease: [0.23, 1, 0.32, 1] }}
                className="h-full flex flex-col items-center justify-center text-center max-w-2xl mx-auto p-8"
              >
                <div className="w-24 h-24 bg-white rounded-[32px] apple-shadow-lg flex items-center justify-center mb-8 border border-gray-100">
                  <MessageSquare size={40} className="text-indigo-600" />
                </div>
                <h1 className="text-4xl font-extrabold text-gray-900 mb-4 tracking-tight">O quadro está vazio.</h1>
                <p className="text-gray-500 text-lg mb-10 leading-relaxed font-medium">
                  O ClawFlow utiliza o Modo Planejador para transformar suas ideias em POPs e tarefas. Qual é a visão para o seu novo projeto hoje?
                </p>
                
                <button 
                  onClick={handleStartProject}
                  className="bg-indigo-600 hover:bg-indigo-700 text-white px-8 py-4 rounded-[24px] font-bold text-lg transition-all apple-shadow-lg flex items-center space-x-3 active:scale-95 mb-12"
                >
                  <Plus size={24} />
                  <span>Iniciar Novo Projeto</span>
                </button>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 w-full">
                  <QuickActionCard title="Novo Módulo Totvs" subtitle="Desenvolvimento de Software" onClick={handleStartProject} />
                  <QuickActionCard title="Infra de Rede" subtitle="Operação e Hardware" onClick={handleStartProject} />
                </div>
              </motion.div>
            ) : (
              <motion.div 
                key="chat"
                initial={{ opacity: 0, y: 40 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 20 }}
                transition={{ duration: 0.5, ease: [0.23, 1, 0.32, 1] }}
                className="h-full flex flex-col p-8 max-w-4xl mx-auto"
              >
                {/* Chat History Area */}
                <div className="flex-1 space-y-6 overflow-y-auto mb-6 pr-4">
                  {chatHistory.map((item, idx) => (
                    <ChatBubble key={idx} sender={item.sender as 'agent' | 'user'} message={item.message} />
                  ))}
                </div>

                {/* Input Area */}
                <div className="relative group">
                  <input 
                    type="text"
                    value={message}
                    onChange={(e) => setMessage(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
                    placeholder="Descreva seu projeto aqui..."
                    className="w-full bg-white border border-gray-200 rounded-[28px] py-5 px-8 pr-16 apple-shadow-lg focus:outline-none focus:border-indigo-500 transition-all text-lg font-medium"
                  />
                  <button 
                    onClick={handleSendMessage}
                    className="absolute right-4 top-1/2 -translate-y-1/2 w-12 h-12 bg-indigo-600 rounded-full flex items-center justify-center text-white shadow-md active:scale-90 transition-transform hover:bg-indigo-700"
                  >
                    <Send size={20} />
                  </button>
                </div>
                <p className="text-center text-[10px] text-gray-400 font-black uppercase tracking-widest mt-4">
                  Modo Planejador Ativo • Sessão: Agent Main
                </p>
              </motion.div>
            )}
          </AnimatePresence>
        </section>
      </main>
    </div>
  );
};

const NavItem = ({ icon, label, active = false, onClick }: NavItemProps) => (
  <div 
    onClick={onClick}
    className={`flex items-center space-x-3 px-4 py-3 rounded-2xl cursor-pointer transition-all duration-200 ${
      active ? 'bg-indigo-600 text-white apple-shadow' : 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'
    }`}
  >
    {icon}
    <span className="font-bold text-sm hidden md:block">{label}</span>
  </div>
);

const ChatBubble = ({ sender, message }: { sender: 'agent' | 'user', message: string }) => (
  <motion.div 
    initial={{ opacity: 0, x: sender === 'user' ? 20 : -20 }}
    animate={{ opacity: 1, x: 0 }}
    className={`flex ${sender === 'user' ? 'justify-end' : 'justify-start'} items-end space-x-3`}
  >
    {sender === 'agent' && (
      <div className="w-8 h-8 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-600 mb-1">
        <Bot size={18} />
      </div>
    )}
    <div className={`max-w-[80%] p-6 rounded-[28px] text-lg font-medium apple-shadow ${
      sender === 'user' ? 'bg-indigo-600 text-white rounded-br-none' : 'bg-white text-gray-900 rounded-bl-none border border-gray-50'
    }`}>
      {message}
    </div>
  </motion.div>
);

const QuickActionCard = ({ title, subtitle, onClick }: { title: string, subtitle: string, onClick: () => void }) => (
  <div 
    onClick={onClick}
    className="bg-white p-6 rounded-[24px] border border-gray-200 text-left hover:border-indigo-500 hover:apple-shadow-lg transition-all cursor-pointer group active:scale-[0.98]"
  >
    <h4 className="font-bold text-gray-900 group-hover:text-indigo-600 transition-colors">{title}</h4>
    <p className="text-[10px] text-gray-400 font-black uppercase tracking-widest mt-1">{subtitle}</p>
  </div>
);

export default Dashboard;
