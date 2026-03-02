import React, { useEffect, useRef } from 'react';
import { Message } from '../types';
import { ActionForm } from './ActionForm';
import { ActionPreview } from './ActionPreview';

interface MessageListProps {
  messages: Message[];
  onFormSubmit?: (messageId: string, formData: Record<string, any>) => void;
  onActionConfirm?: (messageId: string) => void;
  onActionCancel?: (messageId: string) => void
}

export const MessageList: React.FC<MessageListProps> = ({
  messages,
  onFormSubmit,
  onActionConfirm,
  onActionCancel,
}) => {
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // 自动滚动到底部
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  // 当消息变化时滚动到底部
  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // 组件加载时滚动到底部
  useEffect(() => {
    scrollToBottom();
  }, []);

  return (
    <div className="flex-1 overflow-y-auto bg-white rounded-xl border border-gray-200 shadow-sm p-4">
      <div className="flex flex-col gap-4">
        {messages.map((msg) => (
          <div key={msg.id}>
            {/* Model badge for assistant messages */}
            {msg.role === 'assistant' && msg.model && (
              <div className="text-xs text-gray-400 mb-1 ml-2">
                {msg.model}
              </div>
            )}

            {/* Main message content */}
            <div
              className={`max-w-[75%] p-3 px-4.5 rounded-2xl leading-relaxed shadow-sm ${
                msg.role === 'user'
                  ? 'self-end bg-indigo-600 text-white rounded-br-sm ml-auto'
                  : msg.role === 'system'
                  ? 'self-center bg-transparent text-gray-500 text-center text-sm shadow-none max-w-[90%]'
                  : 'self-start bg-gray-100 text-gray-900 rounded-bl-sm mr-auto'
              }`}
            >
              <p className="whitespace-pre-wrap break-words">{msg.content}</p>
            </div>

            {/* Clarification Form */}
            {msg.clarification && (
              <div className="mt-3 ml-0">
                <ActionForm
                  clarification={msg.clarification}
                  onSubmit={(formData) => onFormSubmit?.(msg.id, formData)}
                  onCancel={() => onActionCancel?.(msg.id)}
                />
              </div>
            )}

            {/* Action Preview */}
            {msg.actionPreview && (
              <div className="mt-3 ml-0">
                <ActionPreview
                  preview={msg.actionPreview}
                  onConfirm={() => onActionConfirm?.(msg.id)}
                  onCancel={() => onActionCancel?.(msg.id)}
                />
              </div>
            )}
          </div>
        ))}
        {/* 滚动锚点 */}
        <div ref={messagesEndRef} />
      </div>
    </div>
  );
};