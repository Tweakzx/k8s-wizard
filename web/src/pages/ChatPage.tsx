import React, { useState } from 'react';
import { NavItem, QuickAction } from '../types';
import { Header } from '../components/Header';
import { Sidebar } from '../components/Sidebar';
import { MessageList } from '../components/MessageList';
import { QuickActions } from '../components/QuickActions';
import { ChatInput } from '../components/ChatInput';
import { useMessages } from '../hooks/useMessages';
import { useConnectionStatus } from '../hooks/useConnectionStatus';

export const ChatPage: React.FC = () => {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [loading, setLoading] = useState(false);
  const [pendingContent, setPendingContent] = useState<string>('');
  const { messages, addMessage, updateMessage } = useMessages();
  const connected = useConnectionStatus();

  const navItems: NavItem[] = [
    { id: 'chat', label: '聊天', icon: '💬' },
    { id: 'resources', label: '资源', icon: '📦' },
    { id: 'history', label: '历史', icon: '📜' },
    { id: 'settings', label: '设置', icon: '⚙️' },
  ];

  const [activeNav, setActiveNav] = useState('chat');

  const quickActions: QuickAction[] = [
    { label: '部署 nginx', command: '部署一个 nginx' },
    { label: '查看 Pod', command: '查看所有 pod' },
    { label: '扩容 5 个', command: '扩容到 5 个副本' },
    { label: '删除旧 Pod', command: '删除名为 old 的 pod' },
  ];

  const sendMessage = async (content: string, formData?: Record<string, any>, confirm?: boolean) => {
    setLoading(true);

    try {
      const response = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content, formData, confirm }),
      });

      const data = await response.json();
      return data;
    } catch (error) {
      return { error: error instanceof Error ? error.message : '未知错误' };
    } finally {
      setLoading(false);
    }
  };

  const handleSendMessage = async (content: string) => {
    addMessage({
      id: Date.now().toString(),
      role: 'user',
      content,
      timestamp: new Date(),
    });

    const data = await sendMessage(content);

    if (data.error) {
      addMessage({
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: `❌ 错误: ${data.error}`,
        timestamp: new Date(),
        model: data.model,
      });
      return;
    }

    const msgId = (Date.now() + 1).toString();

    if (data.clarification) {
      addMessage({
        id: msgId,
        role: 'assistant',
        content: '',
        clarification: data.clarification,
        timestamp: new Date(),
      });
      // messageId tracked via DOM
      setPendingContent(content);
    } else if (data.actionPreview) {
      addMessage({
        id: msgId,
        role: 'assistant',
        content: '',
        actionPreview: data.actionPreview,
        timestamp: new Date(),
      });
      // messageId tracked via DOM
      setPendingContent(content);
    } else {
      addMessage({
        id: msgId,
        role: 'assistant',
        content: data.result,
        timestamp: new Date(),
        model: data.model,
      });
    }
  };

  const handleFormSubmit = async (messageId: string, formData: Record<string, any>) => {
    if (!pendingContent) return;

    updateMessage(messageId, { clarification: undefined });

    const data = await sendMessage(pendingContent, formData);

    if (data.error) {
      updateMessage(messageId, {
        content: `❌ 错误: ${data.error}`,
      });
      return;
    }

    if (data.actionPreview) {
      updateMessage(messageId, {
        content: '',
        actionPreview: data.actionPreview,
      });
    } else {
      updateMessage(messageId, {
        content: data.result,
        model: data.model,
      });
      // messageId cleared
      setPendingContent('');
    }
  };

  const handleActionConfirm = async (messageId: string) => {
    if (!pendingContent) return;

    const data = await sendMessage(pendingContent, undefined, true);

    if (data.error) {
      updateMessage(messageId, {
        actionPreview: undefined,
        content: `❌ 错误: ${data.error}`,
      });
    } else {
      updateMessage(messageId, {
        actionPreview: undefined,
        content: data.result,
        model: data.model,
      });
    }

    // messageId cleared
    setPendingContent('');
  };

  const handleActionCancel = (messageId: string) => {
    updateMessage(messageId, {
      clarification: undefined,
      actionPreview: undefined,
      content: '❌ 操作已取消',
    });
    // messageId cleared
    setPendingContent('');
  };

  return (
    <div className="h-screen w-screen bg-gray-50 flex flex-col overflow-hidden fixed top-0 left-0 right-0 bottom-0">
      <Header onMenuClick={() => setSidebarCollapsed(!sidebarCollapsed)} connected={connected} />

      <div className="flex-1 flex overflow-hidden">
        <Sidebar
          items={navItems}
          activeId={activeNav}
          onItemClick={setActiveNav}
          collapsed={sidebarCollapsed}
        />

        <main className="flex-1 flex flex-col overflow-hidden p-6 bg-gray-50">
          <MessageList
            messages={messages}
            onFormSubmit={handleFormSubmit}
            onActionConfirm={handleActionConfirm}
            onActionCancel={handleActionCancel}
          />
          <QuickActions actions={quickActions} onActionClick={handleSendMessage} />
          <ChatInput onSend={handleSendMessage} loading={loading} />
        </main>
      </div>
    </div>
  );
};
