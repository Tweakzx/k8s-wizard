import React from 'react';
import { NavItem } from '../types';

interface SidebarProps {
  items: NavItem[];
  activeId: string;
  onItemClick: (id: string) => void;
  collapsed: boolean;
}

export const Sidebar: React.FC<SidebarProps> = ({
  items,
  activeId,
  onItemClick,
  collapsed,
}) => {
  return (
    <aside
      className={`bg-white border-r border-gray-200 transition-all duration-300 flex-shrink-0 ${
        collapsed ? 'w-0 opacity-0 overflow-hidden' : 'w-60'
      }`}
    >
      <nav className="flex flex-col py-5 px-3 gap-1">
        {items.map((item) => (
          <button
            key={item.id}
            onClick={() => onItemClick(item.id)}
            className={`flex items-center gap-3 px-4 py-3 rounded-lg font-medium text-sm transition-all duration-200 text-left ${
              activeId === item.id
                ? 'bg-indigo-100 text-indigo-700'
                : 'text-gray-600 hover:bg-gray-100 hover:text-gray-800'
            }`}
          >
            <span className="text-xl">{item.icon}</span>
            <span className="whitespace-nowrap">{item.label}</span>
          </button>
        ))}
      </nav>
    </aside>
  );
};
