import React, { useState } from 'react';
import { NavItem, QuickAction, Suggestion } from '../types';
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
  const [pendingFormData, setPendingFormData] = useState<Record<string, any> | undefined>();
  const [selectedSuggestion, setSelectedSuggestion] = useState<Suggestion | null>(null);
  const { messages, addMessage, updateMessage, clearMessages } = useMessages();
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
    console.log('📤 发送请求:', { content, formData, confirm });

    try {
      const response = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content, formData, confirm }),
      });

      // 检查响应状态
      if (!response.ok) {
        return { error: `服务器错误: ${response.status}` };
      }

      // 检查响应内容
      const text = await response.text();
      if (!text) {
        return { error: '服务器返回空响应' };
      }

      const data = JSON.parse(text);
      console.log('📥 收到响应:', data);
      return data;
    } catch (error) {
      console.error('❌ 请求错误:', error);
      return { error: error instanceof Error ? error.message : '网络错误，请检查后端服务' };
    } finally {
      setLoading(false);
    }
  };

  const handleSendMessage = async (content: string) => {
    // 处理 /clear 命令
    if (content.trim().toLowerCase() === '/clear') {
      clearMessages();
      setPendingContent('');
      setPendingFormData(undefined);
      return;
    }

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

    if (data.suggestions && data.suggestions.length > 0) {
      // Show suggestions instead of form
      addMessage({
        id: msgId,
        role: 'assistant',
        content: '',
        suggestions: data.suggestions,
        timestamp: new Date(),
      });
      setPendingContent(content);
      setPendingFormData(undefined);
    } else if (data.clarification) {
      addMessage({
        id: msgId,
        role: 'assistant',
        content: '',
        clarification: data.clarification,
        timestamp: new Date(),
      });
      setPendingContent(content);
      setPendingFormData(undefined);
    } else if (data.actionPreview) {
      addMessage({
        id: msgId,
        role: 'assistant',
        content: '',
        actionPreview: data.actionPreview,
        timestamp: new Date(),
      });
      setPendingContent(content);
      setPendingFormData(undefined);
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

    // Include selected suggestion ID for tracking
    const payload = {
      content: pendingContent,
      formData: {
        ...formData,
        suggestionId: selectedSuggestion?.id,
      },
    };

    // 保存 formData 以便确认时使用
    setPendingFormData(payload.formData);

    const data = await sendMessage(pendingContent, payload.formData);

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
      setPendingContent('');
      setPendingFormData(undefined);
      setSelectedSuggestion(null);
    }
  };

  const handleActionConfirm = async (messageId: string) => {
    if (!pendingContent) return;

    console.log('✅ 确认执行, pendingFormData:', pendingFormData);

    // 使用保存的 formData 发送确认请求
    const data = await sendMessage(pendingContent, pendingFormData, true);

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

    setPendingContent('');
    setPendingFormData(undefined);
  };

  const handleActionCancel = (messageId: string) => {
    updateMessage(messageId, {
      clarification: undefined,
      actionPreview: undefined,
      content: '❌ 操作已取消',
    });
    setPendingContent('');
    setPendingFormData(undefined);
    setSelectedSuggestion(null);
  };

  const handleSuggestionSelect = (suggestion: Suggestion) => {
    setSelectedSuggestion(suggestion);

    // Auto-populate form fields from suggestion
    const newFormData: Record<string, any> = {
      name: suggestion.name,
      namespace: suggestion.namespace,
    };

    setPendingFormData(newFormData);
  };

  const handleSuggestionNone = () => {
    setSelectedSuggestion(null);
    setPendingFormData({});
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
            onSuggestionSelect={handleSuggestionSelect}
            onSuggestionNone={handleSuggestionNone}
          />
          <QuickActions actions={quickActions} onActionClick={handleSendMessage} />
          <ChatInput onSend={handleSendMessage} loading={loading} />
        </main>
      </div>
    </div>
  );
};
