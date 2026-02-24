import { motion } from 'framer-motion';
import { LayoutDashboard, MessageSquare, Settings, Activity, Plus } from 'lucide-react';

interface NavItemProps {
  icon: React.ReactNode;
  label: string;
  active?: boolean;
}

interface QuickActionCardProps {
  title: string;
  subtitle: string;
}

const Dashboard = () => {
  return (
    <div className="flex h-screen w-full bg-[#F5F5F7] overflow-hidden">
      {/* Sidebar - Apple Style */}
      <aside className="w-20 md:w-64 glass border-r border-gray-200 flex flex-col p-4 z-20">
        <div className="flex items-center space-x-3 px-2 mb-10 mt-2">
          <div className="w-10 h-10 bg-indigo-600 rounded-xl flex items-center justify-center text-white font-black text-xl shadow-lg shadow-indigo-200">
            L
          </div>
          <span className="font-extrabold text-xl tracking-tighter hidden md:block">ClawFlow</span>
        </div>

        <nav className="flex-1 space-y-2">
          <NavItem icon={<LayoutDashboard size={20} />} label="Dashboard" active />
          <NavItem icon={<MessageSquare size={20} />} label="Projetos" />
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
          <h2 className="text-2xl font-bold tracking-tight text-gray-900">Visão Geral</h2>
          
          <div className="flex items-center space-x-6">
            <div className="flex items-center space-x-2">
              <span className="relative flex h-3 w-3">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
              </span>
              <span className="text-sm font-bold text-emerald-600 uppercase tracking-widest">Gateway Online</span>
            </div>
            
            <button className="bg-indigo-600 hover:bg-indigo-700 text-white px-5 py-2.5 rounded-2xl font-bold text-sm transition-all apple-shadow-lg flex items-center space-x-2 active:scale-95">
              <Plus size={18} />
              <span>Novo Projeto</span>
            </button>
          </div>
        </header>

        {/* Kanban / Tabula Rasa Area */}
        <section className="flex-1 p-8 overflow-y-auto">
          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, ease: [0.23, 1, 0.32, 1] }}
            className="h-full flex flex-col items-center justify-center text-center max-w-2xl mx-auto"
          >
            <div className="w-24 h-24 bg-white rounded-[32px] apple-shadow-lg flex items-center justify-center mb-8">
              <MessageSquare size={40} className="text-indigo-600" />
            </div>
            <h1 className="text-4xl font-extrabold text-gray-900 mb-4 tracking-tight">O quadro está vazio.</h1>
            <p className="text-gray-500 text-lg mb-10 leading-relaxed font-medium">
              O ClawFlow utiliza o Modo Planejador para transformar suas ideias em POPs e tarefas. Qual é a visão para o seu novo projeto hoje?
            </p>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 w-full">
              <QuickActionCard title="Novo Módulo Totvs" subtitle="Desenvolvimento de Software" />
              <QuickActionCard title="Infra de Rede" subtitle="Operação e Hardware" />
            </div>
          </motion.div>
        </section>
      </main>
    </div>
  );
};

const NavItem = ({ icon, label, active = false }: NavItemProps) => (
  <div className={`flex items-center space-x-3 px-4 py-3 rounded-2xl cursor-pointer transition-all duration-200 ${
    active ? 'bg-indigo-600 text-white apple-shadow' : 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'
  }`}>
    {icon}
    <span className="font-bold text-sm hidden md:block">{label}</span>
  </div>
);

const QuickActionCard = ({ title, subtitle }: QuickActionCardProps) => (
  <div className="bg-white p-6 rounded-[24px] border border-gray-200 text-left hover:border-indigo-500 hover:apple-shadow-lg transition-all cursor-pointer group active:scale-[0.98]">
    <h4 className="font-bold text-gray-900 group-hover:text-indigo-600 transition-colors">{title}</h4>
    <p className="text-xs text-gray-400 font-bold uppercase tracking-widest mt-1">{subtitle}</p>
  </div>
);

export default Dashboard;
