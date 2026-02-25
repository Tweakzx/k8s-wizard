import React from 'react';
import { ModelSelector } from './ModelSelector';

interface HeaderProps {
  onMenuClick: () => void;
  connected: boolean;
}

export const Header: React.FC<HeaderProps> = ({ onMenuClick, connected }) => {
  return (
    <header className="bg-white border-b border-gray-200 shadow-sm flex-shrink-0">
      <div className="flex items-center gap-4 px-6 py-4">
        <button
          onClick={onMenuClick}
          className="p-2 bg-gray-100 rounded-lg text-gray-600 hover:bg-gray-200 hover:text-gray-800 transition-all duration-200"
          title="打开/关闭导航"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <line x1="3" y1="6" x2="21" y2="6" strokeWidth="2" strokeLinecap="round"/>
            <line x1="3" y1="12" x2="21" y2="12" strokeWidth="2" strokeLinecap="round"/>
            <line x1="3" y1="18" x2="21" y2="18" strokeWidth="2" strokeLinecap="round"/>
          </svg>
        </button>

        <div className="flex items-center gap-3 flex-1">
          <span className="text-2xl">🧙</span>
          <span className="text-xl font-bold text-gray-800">
            K8s Wizard
          </span>
        </div>

        {/* Model Selector */}
        <ModelSelector />

        {/* Connection Status */}
        <div className="flex items-center gap-2 px-3 py-1.5 bg-gray-100 rounded-full">
          <span
            className={`w-2 h-2 rounded-full transition-all duration-300 ${
              connected ? 'bg-green-500' : 'bg-red-500'
            }`}
          />
          <span className="text-sm font-medium text-gray-600">
            {connected ? '已连接' : '未连接'}
          </span>
        </div>
      </div>
    </header>
  );
};
