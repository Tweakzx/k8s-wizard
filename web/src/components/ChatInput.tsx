import React, { useState } from 'react';

interface ChatInputProps {
  onSend: (message: string) => Promise<void>;
  loading: boolean;
}

export const ChatInput: React.FC<ChatInputProps> = ({ onSend, loading }) => {
  const [input, setInput] = useState('');

  const handleSubmit = async () => {
    if (!input.trim() || loading) return;
    const message = input.trim();
    setInput('');
    await onSend(message);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div className="flex gap-3 p-5 bg-white rounded-xl border border-gray-200">
      <textarea
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="用自然语言描述你想要的操作..."
        disabled={loading}
        className="flex-1 min-h-[120px] p-4 rounded-lg border-2 border-gray-200 text-gray-900 text-base resize-none focus:outline-none focus:border-indigo-600 focus:ring-3 focus:ring-indigo-100 disabled:opacity-60 disabled:cursor-not-allowed placeholder:text-gray-400"
      />
      <button
        onClick={handleSubmit}
        disabled={loading || !input.trim()}
        className="px-8 py-4 bg-indigo-600 text-white font-semibold text-base rounded-lg hover:bg-indigo-700 hover:-translate-y-0.5 hover:shadow-lg disabled:bg-indigo-300 disabled:cursor-not-allowed disabled:hover:translate-y-0 disabled:hover:shadow-none transition-all duration-200 min-w-[100px]"
      >
        {loading ? '发送中...' : '发送'}
      </button>
    </div>
  );
};
