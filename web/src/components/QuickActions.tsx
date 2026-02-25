import React from 'react';
import { QuickAction } from '../types';

interface QuickActionsProps {
  actions: QuickAction[];
  onActionClick: (command: string) => void;
}

export const QuickActions: React.FC<QuickActionsProps> = ({ actions, onActionClick }) => {
  return (
    <div className="flex items-center gap-3 flex-wrap mb-4">
      <span className="text-sm font-medium text-gray-600">💡 试试说：</span>
      {actions.map((action) => (
        <button
          key={action.label}
          onClick={() => onActionClick(action.command)}
          className="px-4 py-2 bg-white border border-gray-200 rounded-full text-sm text-gray-700 hover:bg-indigo-600 hover:border-indigo-600 hover:text-white hover:-translate-y-px transition-all duration-200"
        >
          {action.label}
        </button>
      ))}
    </div>
  );
};
