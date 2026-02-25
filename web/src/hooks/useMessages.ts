import { useState, useEffect } from 'react';
import { Message } from '../types';

const STORAGE_KEY = 'k8s-wizard-messages';

export function useMessages() {
  const [messages, setMessages] = useState<Message[]>([]);

  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        setMessages(
          parsed.map((m: any) => ({
            ...m,
            timestamp: new Date(m.timestamp),
          }))
        );
      } catch (e) {
        console.error('Failed to load messages:', e);
        loadDefaultMessages();
      }
    } else {
      loadDefaultMessages();
    }
  }, []);

  useEffect(() => {
    if (messages.length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(messages));
    }
  }, [messages]);

  const loadDefaultMessages = () => {
    const defaultMessages: Message[] = [
      {
        id: '1',
        role: 'system',
        content: '🧙 你好！我是 K8s Wizard，可以通过自然语言帮你管理 Kubernetes 集群。',
        timestamp: new Date(),
      },
      {
        id: '2',
        role: 'system',
        content: '💡 试试说："部署一个 nginx"、"查看所有 pod"、"扩容到 5 个副本"',
        timestamp: new Date(),
      },
    ];
    setMessages(defaultMessages);
  };

  const addMessage = (message: Message) => {
    setMessages((prev) => [...prev, message]);
  };

  const updateMessage = (id: string, updates: Partial<Message>) => {
    setMessages((prev) =>
      prev.map((msg) => (msg.id === id ? { ...msg, ...updates } : msg))
    );
  };

  return { messages, addMessage, updateMessage };
}
